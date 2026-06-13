/*
=== AI Agent 概念：向量检索（Vector Retrieval）===

向量检索是 RAG 的核心步骤。将用户问题转化为向量，在向量数据库中
找到语义最相近的文档片段，作为 LLM 回答的参考资料。

核心原理：
  - 文本 → Embedding 模型 → 高维向量（数字数组）
  - 向量之间的距离 = 文本的语义相似度
  - 在 Milvus 中搜索与问题向量最近的文档向量 → 返回相关文档

本文件的角色：
  创建 Milvus 向量检索器实例，作为 DAG 图中的 MilvusRetriever 节点。
  这是一个委托层，实际检索逻辑在 internal/ai/retriever 包中。

关键数据流：
  输入：用户问题文本（string）
  处理：文本 → 向量化 → Milvus 检索 → 返回文档片段
  输出：[]schema.Document（相关文档列表，输出 key 为 "documents"）

关联文件：
  - internal/ai/retriever/retriever.go — Milvus 检索器的实际实现
  - orchestration.go — 将本检索器加入 DAG 图，设置 output key 为 "documents"
  - prompt.go — 使用 {documents} 占位符接收检索结果
*/
package chat_pipeline

import (
	retriever2 "OpsPilot/internal/ai/retriever"
	"context"

	"github.com/cloudwego/eino/components/retriever"
)

// newRetriever 创建 Milvus 向量检索器
//
// 【AI 概念】向量检索的工作流程：
//   1. 用户问题 → Embedding 模型转为向量
//   2. 在 Milvus 中搜索 TopK 个最相似的文档片段
//   3. 返回文档片段列表（包含内容和元数据）
//
// 检索结果通过 compose.WithOutputKey("documents") 标记，
// 这样 prompt.go 中的 {documents} 占位符就能匹配到这些文档
func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	return retriever2.NewMilvusRetriever(ctx)
}
