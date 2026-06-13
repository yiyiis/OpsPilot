# RAG Chat Pipeline 模块

## 概述

本模块实现了 OpsPilot 的 **RAG 对话 Agent**——基于检索增强生成（Retrieval-Augmented Generation）的智能对话系统。

核心思路：用户提问 → 从 Milvus 知识库中检索相关文档 → 将文档注入 Prompt → LLM 基于文档内容生成回答。

## 核心概念

### RAG（检索增强生成）
LLM 的知识来自训练数据，无法获取私有知识库内容。RAG 通过在生成前先检索相关文档，让 LLM 基于真实文档回答，大幅减少幻觉。

### ReAct Agent（推理 + 行动）
ReAct 模式让 LLM 自主决定是否需要调用工具。每轮循环：推理当前情况 → 选择并调用工具 → 观察结果 → 继续推理或输出回答。

### Graph/DAG 编排
Eino 框架用有向无环图（DAG）编排处理步骤。每个节点是一个处理单元，节点之间通过边连接定义数据流向。

## 图节点流程

```
START
  ├──→ InputToRag (提取用户问题文本)
  │       ↓
  │    MilvusRetriever (向量检索相关文档)
  │       ↓
  └──→ InputToChat (提取对话上下文)  ──→  ChatTemplate (组装 Prompt)
                                            ↓
                                       ReActAgent (LLM 推理 + 工具调用)
                                            ↓
                                          END
```

**注意：** `ChatTemplate` 使用 `AllPredecessor` 触发模式——必须等 `MilvusRetriever` 和 `InputToChat` 两条分支都完成后才会执行。

## 文件清单

| 文件 | 职责 | 核心 AI 概念 |
|------|------|-------------|
| `orchestration.go` | 构建 DAG 图，定义节点和边 | Agent 编排（Graph/DAG） |
| `flow.go` | 构建 ReAct Agent，绑定工具和模型 | ReAct Agent |
| `prompt.go` | 定义系统 Prompt 模板，注入 RAG 文档 | RAG Prompt 工程 |
| `retriever.go` | 创建 Milvus 向量检索器 | 向量检索 |
| `embedding.go` | 创建文本向量化模型（当前未接入图） | 文本向量化（Embedding） |
| `lambda_func.go` | 图节点间的数据转换函数 | 数据转换 Lambda |
| `tools_node.go` | DuckDuckGo 搜索工具（当前未启用） | 工具注册 |
| `types.go` | 定义 Pipeline 输入类型 `UserMessage` | 数据类型定义 |
| `model.go` | 创建 LLM 模型实例（委托给 models 包） | LLM 模型 |
