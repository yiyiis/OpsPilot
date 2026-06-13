/*
=== AI Agent 概念：向量索引写入（Indexing）===

向量索引写入是知识索引 Pipeline 的最后一步。
将分块后的文档片段向量化并存入 Milvus 向量数据库。

核心原理：
  - 文档片段 → Embedding 模型 → 向量
  - 向量 + 原文 + 元数据 → 写入 Milvus
  - 写入后即可在 RAG 检索时被匹配到

本文件的角色：
  创建 Milvus 索引写入器实例，作为 DAG 图的最后一个节点。
  这是一个委托层，实际实现在 internal/ai/indexer 包中。

关联文件：
  - orchestration.go — 将本索引器加入 DAG 图
  - transformer.go — 上游节点，提供分块后的文档
  - internal/ai/indexer/indexer.go — Milvus 索引器的实际实现
*/
package knowledge_index_pipeline

import (
	indexer2 "OpsPilot/internal/ai/indexer"
	"context"

	"github.com/cloudwego/eino/components/indexer"
)

// newIndexer 创建 Milvus 向量索引写入器
//
// 【AI 概念】索引写入流程
// MilvusIndexer 内部会：
//   1. 对每个文档片段调用 Embedding 模型生成向量
//   2. 将向量 + 原文内容 + 元数据写入 Milvus 集合
//   3. 返回每个文档片段的 ID 列表
//
// 输入：分块后的文档片段（[]document.Document）
// 输出：文档片段 ID 列表（[]string）
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	return indexer2.NewMilvusIndexer(ctx)
}
