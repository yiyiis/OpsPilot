/*
=== AI Agent 概念：知识索引流水线编排 ===

知识索引是 RAG 的前置步骤：将文档转化为向量并存入向量数据库。
本文件用线性 DAG 图编排三个处理节点：文件加载 → 分块 → 写入 Milvus。

核心原理：
  - 使用 Eino 的 compose.NewGraph 创建线性图
  - 三个节点按顺序执行（与 Chat Pipeline 的并行图不同）
  - 使用 AnyPredecessor 触发模式（线性图中每个节点只有一个前驱）

本文件的角色：
  定义知识索引的完整 DAG 图拓扑。调用 BuildKnowledgeIndexing 即可
  得到一个可运行的索引 Pipeline。

关键数据流（线性流水线）：
  文件源(document.Source)
    → FileLoader（加载文件内容）
    → MarkdownSplitter（按标题分块）
    → MilvusIndexer（向量化 + 写入 Milvus）
    → []string（输出每个文档片段的 ID 列表）

关联文件：
  - loader.go — FileLoader 节点实现
  - transformer.go — MarkdownSplitter 节点实现
  - indexer.go — MilvusIndexer 节点实现
  - embedding.go — Embedding 模型（当前未接入图）
*/
package knowledge_index_pipeline

import (
	"context"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

// BuildKnowledgeIndexing 构建知识索引的线性 DAG 图并编译为可执行对象
//
// 【AI 概念】线性图 vs 并行图
// 与 Chat Pipeline 的并行分支图不同，知识索引是简单的线性流水线：
//   START → FileLoader → MarkdownSplitter → MilvusIndexer → END
//
// 每个节点的输出就是下一个节点的输入，没有分支和汇聚。
// 因此使用 AnyPredecessor 触发模式（任一前驱完成即触发）。
//
// 返回值：compose.Runnable[document.Source, []string]
//   - 输入：document.Source（文件路径等）
//   - 输出：[]string（索引后的文档片段 ID 列表）
func BuildKnowledgeIndexing(ctx context.Context) (r compose.Runnable[document.Source, []string], err error) {
	// 定义 3 个图节点的名称常量
	const (
		FileLoader       = "FileLoader"       // 文件加载节点
		MarkdownSplitter = "MarkdownSplitter" // Markdown 分块节点
		MilvusIndexer    = "MilvusIndexer"    // Milvus 索引写入节点
	)

	// 创建线性图
	g := compose.NewGraph[document.Source, []string]()

	// === 添加节点 ===

	// 节点1：FileLoader — 加载文件内容为 Eino document 对象
	fileLoaderKeyOfLoader, err := newLoader(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(FileLoader, fileLoaderKeyOfLoader)

	// 节点2：MarkdownSplitter — 按 Markdown 标题分块
	markdownSplitterKeyOfDocumentTransformer, err := newDocumentTransformer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(MarkdownSplitter, markdownSplitterKeyOfDocumentTransformer)

	// 节点3：MilvusIndexer — 向量化并写入 Milvus 向量数据库
	milvusIndexerKeyOfIndexer, err := newIndexer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(MilvusIndexer, milvusIndexerKeyOfIndexer)

	// === 定义边（线性顺序）===
	_ = g.AddEdge(compose.START, FileLoader)        // 开始 → 加载文件
	_ = g.AddEdge(MilvusIndexer, compose.END)        // 索引完成 → 结束
	_ = g.AddEdge(FileLoader, MarkdownSplitter)      // 加载 → 分块
	_ = g.AddEdge(MarkdownSplitter, MilvusIndexer)   // 分块 → 索引写入

	// 编译图
	// AnyPredecessor：线性图中每个节点只有一个前驱，所以任一前驱完成即触发
	r, err = g.Compile(ctx, compose.WithGraphName("KnowledgeIndexing"), compose.WithNodeTriggerMode(compose.AnyPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
