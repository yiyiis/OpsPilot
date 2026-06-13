/*
=== AI Agent 概念：Embedding 模型封装 ===

Embedding（文本向量化）是将文本转化为高维数字数组的过程。
本文件封装了 Embedding 模型的创建逻辑，供 RAG 检索器和知识索引 Pipeline 使用。

核心原理：
  - 输入一段文本 → 输出一个高维向量（数字数组）
  - 语义相似的文本 → 向量距离近（可以被检索到）
  - 语义不同的文本 → 向量距离远（不会被检索到）

本文件的角色：
  提供统一的 Embedding 模型创建入口。所有需要文本向量化的地方
  （RAG 检索、知识索引）都通过此文件获取 Embedding 实例。

技术选型：
  - 模型：Google Gemini embedding-2
  - 输出维度：3072（默认）

关联文件：
  - gemini.go — Gemini Embedder 的具体实现
  - internal/ai/retriever/retriever.go — RAG 检索器使用 Embedding 做向量化
  - internal/ai/indexer/indexer.go — 知识索引器使用 Embedding 做向量化
*/
package embedder

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/embedding"
)

// NewEmbedder 创建文本向量化模型
//
// 使用 Google Gemini embedding-2 模型，输出 3072 维向量。
// 配置项在 config.yaml 的 gemini_embedding_model 下。
func NewEmbedder(ctx context.Context) (eb embedding.Embedder, err error) {
	embedder, err := NewGeminiEmbedder(ctx)
	if err != nil {
		log.Printf("创建 Gemini Embedder 失败: %v\n", err)
		return nil, err
	}
	return embedder, nil
}
