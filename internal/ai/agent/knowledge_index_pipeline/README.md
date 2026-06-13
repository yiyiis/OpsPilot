# Knowledge Index Pipeline 模块

## 概述

本模块实现了 OpsPilot 的 **知识索引 Pipeline**——将文档转化为向量并存入 Milvus 向量数据库。

核心思路：文件 → 加载 → 按 Markdown 标题分块 → 向量化 → 写入 Milvus。这是 RAG 的前置步骤：没有索引好的知识库，就无法进行检索增强生成。

## 核心概念

### 知识索引流程

```
文件上传 (.md / .txt)
    ↓
FileLoader（加载文件内容）
    ↓
MarkdownSplitter（按 H1 标题分块）
    ↓
MilvusIndexer（向量化 + 写入 Milvus）
```

### 文档分块（Chunking）

为什么需要分块？因为：
1. 完整文档太长，超出 LLM 上下文窗口
2. 检索时需要精确匹配，整篇文档粒度太粗
3. 分块后的文档片段可以独立被检索和注入 Prompt

本项目使用 **Markdown Header Splitter**——按 `#`（H1 标题）分割，每个章节成为一个独立的文档片段。

### 向量化（Embedding）

文本需要转化为向量（高维数字数组）才能进行语义检索。本项目使用 Google Gemini 的 `embedding-2` 模型。

## 文件清单

| 文件 | 职责 | 行数 |
|------|------|------|
| `orchestration.go` | 构建线性图编排（Loader → Splitter → Indexer） | 42 |
| `loader.go` | 创建文件加载器 | 19 |
| `transformer.go` | 创建 Markdown 标题分割器 | 27 |
| `indexer.go` | 创建 Milvus 索引写入器 | 13 |
| `embedding.go` | 创建 Embedding 模型（当前未接入图） | 12 |

## 注意事项

- `embedding.go` 中定义了 `newEmbedding` 函数，但当前 **未被图编排使用**。Milvus Indexer 内部可能自行处理了向量化。
- `indexer.go` 中有一条旧注释提到了 "RedisIndexer"，但实际使用的是 MilvusIndexer。
