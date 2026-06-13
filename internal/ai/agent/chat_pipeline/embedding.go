/*
=== AI Agent 概念：文本向量化（Embedding）===

Embedding（文本向量化）是将文本转化为高维数字数组的过程。
语义相近的文本会被映射到向量空间中相近的位置，使得向量检索成为可能。

核心原理：
  - 输入一段文本 → Embedding 模型 → 输出一个高维向量（如 65536 维）
  - 语义相似的文本 → 向量距离近（余弦相似度高）
  - 语义不同的文本 → 向量距离远（余弦相似度低）

本文件的角色：
  创建 Embedding 模型实例。这是一个委托层，实际实现在 internal/ai/embedder 包中。
  注意：当前这个函数【未被 DAG 图直接使用】——Milvus Retriever 内部会自行处理向量化。

关键数据流：
  输入：文本字符串
  处理：文本 → Google Gemini embedding-2 → 向量
  输出：高维向量（dim=3072）

关联文件：
  - internal/ai/embedder/embedder.go — Embedding 模型的实际实现
  - internal/ai/agent/knowledge_index_pipeline/embedding.go — 索引 Pipeline 中的同名函数
*/
package chat_pipeline

import (
	"OpsPilot/internal/ai/embedder"
	"context"

	"github.com/cloudwego/eino/components/embedding"
)

// newEmbedding 创建文本向量化模型
//
// 【AI 概念】Embedding 模型选择
// 使用 Google Gemini embedding-2 模型
//
// 注意：此函数当前未被 orchestration.go 中的图使用。
// Milvus Retriever 在内部自行完成向量化。
// 此函数可能预留给未来需要在图层面单独使用 Embedding 的场景。
func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	return embedder.NewEmbedder(ctx)
}
