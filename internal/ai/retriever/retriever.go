package retriever

import (
	"OpsPilot/internal/ai/embedder"
	"OpsPilot/utility/client"
	"OpsPilot/utility/common"
	"context"

	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/cloudwego/eino/components/retriever"
)

// cosineSearchParam 为 COSINE 度量创建合理的搜索参数
// Eino 默认用向量维度作为 radius，对 COSINE（范围 0~1）不适用
func cosineSearchParam() entity.SearchParam {
	sp, _ := entity.NewIndexAUTOINDEXSearchParam(1)
	sp.AddRadius(0.0)         // COSINE 距离下界（0 = 完全相同）
	sp.AddRangeFilter(2.0)    // COSINE 距离上界（2 = 完全相反）
	return sp
}

// NewMilvusRetriever 创建 Milvus 向量检索器
//
// 【AI 概念】向量检索的配置参数
//   - Client: Milvus 数据库连接（用于执行检索查询）
//   - Collection: 要检索的集合名称（"biz"）
//   - VectorField: 向量字段名（"vector"）
//   - OutputFields: 检索结果中要返回的字段（排除向量本身，只返回内容）
//   - TopK: 返回最相关的 K 个结果（当前设为 1，只返回最匹配的 1 个）
//   - Embedding: Embedding 模型（用于将查询文本转为向量）
//
// 检索流程：
//  1. 查询文本 → Embedding 模型 → 查询向量
//  2. 在 Milvus 中搜索与查询向量最近的 TopK 个文档向量
//  3. 返回文档片段列表（按相似度排序）
func NewMilvusRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	// 获取 Milvus 客户端连接
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		return nil, err
	}

	// 获取 Embedding 模型（用于将查询文本转为向量）
	eb, err := embedder.NewEmbedder(ctx)
	if err != nil {
		return nil, err
	}

	// 创建 Milvus 检索器
	r, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		Client:      cli,                          // Milvus 客户端
		Collection:  common.MilvusCollectionName,  // 集合名："biz"
		VectorField: "vector",                     // 向量字段名
		OutputFields: []string{                    // 返回的字段（不含向量本身）
			"id",
			"content",
			"metadata",
		},
		TopK:      1,                          // 只返回最相关的 1 个文档片段
		Embedding: eb,                         // Embedding 模型
		MetricType: entity.COSINE,             // FloatVector 使用余弦相似度
		// 自定义向量转换器：float64 → float32，匹配 FloatVector 字段类型
		// Eino 默认的 defaultVectorConverter 转为 BinaryVector，与 FloatVector 不兼容
		VectorConverter: floatVectorConverter,
		Sp:             cosineSearchParam(),
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// floatVectorConverter 将 float64 向量转换为 Milvus FloatVector 类型
func floatVectorConverter(ctx context.Context, vectors [][]float64) ([]entity.Vector, error) {
	result := make([]entity.Vector, 0, len(vectors))
	for _, v := range vectors {
		f32 := make([]float32, len(v))
		for i, val := range v {
			f32[i] = float32(val)
		}
		result = append(result, entity.FloatVector(f32))
	}
	return result, nil
}
