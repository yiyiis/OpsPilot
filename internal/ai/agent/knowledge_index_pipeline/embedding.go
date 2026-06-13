/*
=== AI Agent 概念：文本向量化（Embedding）===

本文件为知识索引 Pipeline 提供 Embedding 模型实例。
当前此函数【未被 DAG 图直接使用】——MilvusIndexer 内部自行处理向量化。

核心原理：
  - 与 chat_pipeline/embedding.go 中的函数完全相同
  - 使用 Google Gemini embedding-2 模型
  - 输出维度：3072

状态：⚠️ 未接入图编排（MilvusIndexer 内部自行调用 Embedding）

关联文件：
  - orchestration.go — DAG 图编排（未使用此函数）
  - internal/ai/embedder/embedder.go — Embedding 模型的实际实现
*/
package knowledge_index_pipeline

import (
	"OpsPilot/internal/ai/embedder"
	"context"

	"github.com/cloudwego/eino/components/embedding"
)

// newEmbedding 创建文本向量化模型
//
// 注意：此函数当前未被 orchestration.go 中的图使用。
// MilvusIndexer 在内部自行完成向量化。
// 保留此函数可能是为了将来在图层面单独控制 Embedding 步骤。
func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	return embedder.NewEmbedder(ctx)
}
