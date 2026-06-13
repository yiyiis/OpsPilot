/*
=== AI Agent 概念：RAG 工具（工具内的 RAG）===

这个工具展示了 Agent 工具如何组合使用 RAG 技术。
当 LLM 需要查询内部运维文档时，调用此工具进行 Milvus 向量检索。

核心原理：
  - 工具本身就是一个 RAG 检索器：接收查询文本 → Milvus 检索 → 返回文档
  - LLM 可以在推理过程中自主决定是否需要查询内部文档
  - 这是"工具组合"的示例：Agent 工具内部封装了另一个 AI 能力（RAG）

本文件的角色：
  提供内部文档 RAG 检索工具。与 Chat Pipeline 主流程的 RAG 不同，
  这个工具是 LLM 在推理过程中按需调用的，而非自动触发的。

关键数据流：
  LLM 推理 → "需要查内部文档" → 调用 query_internal_docs 工具
  → Milvus 检索 → 返回相关文档片段 → LLM 继续推理

关联文件：
  - internal/ai/retriever/retriever.go — Milvus 检索器实现
  - chat_pipeline/retriever.go — 主流程 RAG 检索（对比：自动触发 vs 工具触发）
*/
package tools

import (
	"OpsPilot/internal/ai/retriever"
	"context"
	"encoding/json"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// QueryInternalDocsInput 内部文档查询工具的输入参数
//
// 【AI 概念】简单输入 Schema
// 只需要一个 Query 字段——LLM 生成要搜索的查询文本
type QueryInternalDocsInput struct {
	Query string `json:"query" jsonschema:"description=The query string to search in internal documentation for relevant information and processing steps"`
}

// NewQueryInternalDocsTool 创建内部文档 RAG 检索工具
//
// 【AI 概念】工具内嵌 RAG
// 与 Chat Pipeline 主流程的 RAG 检索不同：
//
//   主流程 RAG（chat_pipeline/retriever.go）：
//     - 自动触发：每个用户请求都会先检索文档
//     - 作为 DAG 图节点存在
//
//   工具 RAG（本文件）：
//     - 按需调用：LLM 判断需要时才调用
//     - 作为 Agent 工具存在
//
// 两种方式互补：主流程 RAG 提供基础上下文，工具 RAG 提供按需深入查询
func NewQueryInternalDocsTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_internal_docs", // 工具名
		"Use this tool to search internal documentation and knowledge base for relevant information. It performs RAG (Retrieval-Augmented Generation) to find similar documents and extract processing steps. This is useful when you need to understand internal procedures, best practices, or step-by-step guides stored in the company's documentation.",
		func(ctx context.Context, input *QueryInternalDocsInput, opts ...tool.Option) (output string, err error) {
			// 创建 Milvus 检索器（每次调用都新建实例）
			rr, err := retriever.NewMilvusRetriever(ctx)
			if err != nil {
				log.Fatal(err)
			}

			// 执行向量检索
			resp, err := rr.Retrieve(ctx, input.Query)
			if err != nil {
				log.Fatal(err)
			}

			// 将检索结果序列化为 JSON 返回给 LLM
			respBytes, _ := json.Marshal(resp)
			output = string(respBytes)
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
