package chat

import (
	"OpsPilot/internal/ai/agent/knowledge_index_pipeline"
	loader2 "OpsPilot/internal/ai/loader"
	"OpsPilot/utility/client"
	"OpsPilot/utility/common"
	"OpsPilot/utility/log_call_back"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
	"github.com/gin-gonic/gin"
)

// FileUpload 上传文件到知识库
// @Summary      文件上传
// @Description  上传 .md/.txt 文件到知识库，自动分块、向量化、入库
// @Tags         upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "上传文件"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /upload [post]
func (h *ChatHandler) FileUpload(c *gin.Context) {
	// 从请求中获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请上传文件", "data": nil})
		return
	}

	// 确保保存目录存在
	if err := os.MkdirAll(common.FileDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("创建目录失败: %v", err), "data": nil})
		return
	}

	// 保存文件
	savePath := filepath.Join(common.FileDir, file.Filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("保存文件失败: %v", err), "data": nil})
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("获取文件信息失败: %v", err), "data": nil})
		return
	}

	// 构建知识索引
	if err := buildIntoIndex(c.Request.Context(), savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("构建知识库失败: %v", err), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": FileUploadRes{
			FileName: file.Filename,
			FilePath: savePath,
			FileSize: fileInfo.Size(),
		},
	})
}

// buildIntoIndex 构建知识索引（先删除旧数据，再重新索引）
func buildIntoIndex(ctx context.Context, path string) error {
	r, err := knowledge_index_pipeline.BuildKnowledgeIndexing(ctx)
	if err != nil {
		return err
	}

	// 删除旧数据
	loader, err := loader2.NewFileLoader(ctx)
	if err != nil {
		return err
	}
	docs, err := loader.Load(ctx, document.Source{URI: path})
	if err != nil {
		return err
	}
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		return err
	}

	// 查询所有 metadata 中 _source 一样的数据并删除
	expr := fmt.Sprintf(`metadata["_source"] == "%s"`, docs[0].MetaData["_source"])
	queryResult, err := cli.Query(ctx, common.MilvusCollectionName, []string{}, expr, []string{"id"})
	if err != nil {
		return err
	} else if len(queryResult) > 0 {
		var idsToDelete []string
		for _, column := range queryResult {
			if column.Name() == "id" {
				for i := 0; i < column.Len(); i++ {
					id, err := column.GetAsString(i)
					if err == nil {
						idsToDelete = append(idsToDelete, id)
					}
				}
			}
		}
		if len(idsToDelete) > 0 {
			deleteExpr := fmt.Sprintf(`id in ["%s"]`, strings.Join(idsToDelete, `","`))
			err = cli.Delete(ctx, common.MilvusCollectionName, "", deleteExpr)
			if err != nil {
				fmt.Printf("[warn] delete existing data failed: %v\n", err)
			} else {
				fmt.Printf("[info] deleted %d existing records with _source: %s\n", len(idsToDelete), docs[0].MetaData["_source"])
			}
		}
	}

	// 重新构建索引
	ids, err := r.Invoke(ctx, document.Source{URI: path}, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		return fmt.Errorf("invoke index graph failed: %w", err)
	}
	fmt.Printf("[done] indexing file: %s, len of parts: %d\n", path, len(ids))
	return nil
}
