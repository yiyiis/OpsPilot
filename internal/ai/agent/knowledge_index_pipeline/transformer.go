/*
=== AI Agent 概念：文档分块（Chunking）===

文档分块是知识索引 Pipeline 的第二步。将完整文档按语义边界分割成
较小的片段，每个片段可以独立被检索和注入 Prompt。

核心原理：
  - 为什么分块？完整文档太长，超出 LLM 上下文窗口；且检索粒度太粗
  - Markdown Header Splitter：按标题层级分块，保持语义完整性
  - 每个片段分配唯一 ID，用于在向量数据库中标识

本文件的角色：
  创建 Markdown Header Splitter 实例，作为 DAG 图的第二个节点。

分块策略：
  - 按 H1 标题（#）分割：每个一级标题下的内容成为一个片段
  - 保留标题文本（TrimHeaders: false）
  - 每个片段用 UUID 作为唯一标识

关联文件：
  - orchestration.go — 将本分割器加入 DAG 图
  - loader.go — 上游节点，提供加载后的文档
  - indexer.go — 下游节点，将分块后的文档写入 Milvus
*/
package knowledge_index_pipeline

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/google/uuid"
)

// newDocumentTransformer 创建 Markdown 标题分割器
//
// 【AI 概念】Markdown Header Splitter 的工作方式
//
// 假设输入文档：
//
//	# 服务宕机处理
//	当服务出现 panic 时...
//	## 排查步骤
//	1. 查看 stderr 日志
//
//	# 高失败率处理
//	当接口失败率超过阈值时...
//
// 输出（按 H1 分割）：
//   - 片段1: "# 服务宕机处理\n当服务出现 panic 时...\n## 排查步骤\n1. 查看 stderr 日志"
//   - 片段2: "# 高失败率处理\n当接口失败率超过阈值时..."
//
// 每个片段都会被独立向量化并存入 Milvus，检索时可以精确匹配到相关章节。
//
// 输入：加载后的文档（[]document.Document）
// 输出：分块后的文档片段（[]document.Document，每个带唯一 ID）
func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	config := &markdown.HeaderConfig{
		// 按 H1（#）标题分块
		// 键 "#" 表示一级标题，值 "title" 是分块后的元数据字段名
		Headers: map[string]string{
			"#": "title",
		},
		// TrimHeaders: false — 保留标题文本在分块内容中
		// 如果设为 true，标题会被从内容中移除
		TrimHeaders: false,
		// IDGenerator — 为每个分块生成唯一标识符
		// 用于在 Milvus 中标识不同的文档片段
		// 参数 originalID（原始文档 ID）和 splitIndex（分块索引）被忽略，直接用 UUID
		IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
			return uuid.New().String()
		},
	}
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
