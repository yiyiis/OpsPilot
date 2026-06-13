/*
Gemini Embedding 实现

实现 eino 的 embedding.Embedder 接口，使用 Google Gemini embedding-2 模型。
将文本转为向量，用于 Milvus 索引和检索。

关联文件：
  - embedder.go — 调用此实现创建 Embedder 实例
  - indexer/indexer.go — 灌入 Milvus 时使用
  - retriever/retriever.go — 检索 Milvus 时使用
*/
package embedder

import (
	"context"
	"fmt"
	"log"

	"OpsPilot/utility/config"

	"github.com/cloudwego/eino/components/embedding"
	"google.golang.org/genai"
)

// GeminiEmbedder 实现 eino 的 Embedder 接口
type GeminiEmbedder struct {
	client *genai.Client
	model  string
	dim    int32
}

// NewGeminiEmbedder 创建 Gemini Embedder 实例
func NewGeminiEmbedder(ctx context.Context) (*GeminiEmbedder, error) {
	apiKey := config.GetString("gemini_embedding_model.api_key")
	model := config.GetString("gemini_embedding_model.model")
	if model == "" {
		model = "gemini-embedding-2"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini 客户端失败: %w", err)
	}

	return &GeminiEmbedder{
		client: client,
		model:  model,
		dim:    3072, // gemini-embedding-2 默认维度
	}, nil
}

// EmbedStrings 实现 embedding.Embedder 接口
// 将一组文本转为对应的向量
func (e *GeminiEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// 构造请求内容
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = genai.NewContentFromText(text, genai.RoleUser)
	}

	// 调用 Gemini EmbedContent（支持批量）
	result, err := e.client.Models.EmbedContent(ctx, e.model, contents, &genai.EmbedContentConfig{})
	if err != nil {
		return nil, fmt.Errorf("Gemini EmbedContent 调用失败: %w", err)
	}

	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("Gemini 返回的 embedding 数量(%d)与请求数量(%d)不匹配",
			len(result.Embeddings), len(texts))
	}

	// 转换 []float32 → []float64
	embeddings := make([][]float64, len(result.Embeddings))
	for i, emb := range result.Embeddings {
		if emb == nil || len(emb.Values) == 0 {
			return nil, fmt.Errorf("第 %d 个文本的 embedding 为空", i)
		}
		embeddings[i] = make([]float64, len(emb.Values))
		for j, v := range emb.Values {
			embeddings[i][j] = float64(v)
		}
	}

	log.Printf("[Gemini Embedding] 成功向量化 %d 条文本，维度: %d", len(texts), len(embeddings[0]))
	return embeddings, nil
}
