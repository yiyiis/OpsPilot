# AI Agent 学习注释设计文档

**日期：** 2026-06-12
**目标：** 为 OpsPilot 项目添加分层中文注释和文档，帮助有一定 LLM 基础的开发者学习 AI Agent 核心技术

---

## 背景与目标

OpsPilot 是一个功能完整的 AI OnCall 助手，涵盖了 RAG、ReAct Agent、Plan-Execute-Replan、Tool Use、SSE 流式输出等 AI Agent 核心技术。本项目旨在通过分层注释和文档，将这些技术概念与代码实现对应起来，形成一份可读的学习材料。

**目标读者：** 有 LLM API 调用和 prompt engineering 基础，但对复杂 Agent 模式（编排、工具调用、重规划等）不熟悉的开发者。

---

## 方案概述

采用 **分层注释 + 架构文档** 的方案，分三个层次：

| 层次 | 产出物 | 目的 |
|------|--------|------|
| 顶层架构文档 | `docs/ai-agent-architecture-guide.md` | 全局视角，数据流，概念索引 |
| 模块级文档 | 各模块目录的 `README.md` | 模块职责和核心概念说明 |
| 代码文件注释 | 关键代码文件内的中文注释 | 逐函数/逐行解释概念和逻辑 |

---

## 层次 1：顶层架构文档

**文件：** `docs/ai-agent-architecture-guide.md`

**内容结构：**

1. **项目概述** — OpsPilot 是什么，解决什么问题
2. **系统架构总览** — ASCII 架构图，展示前后端和外部依赖关系
3. **核心 Pipeline 1：RAG Chat**
   - 数据流图：用户问题 → 向量检索 → 文档注入 → ReAct Agent → 响应
   - 涉及文件链接
4. **核心 Pipeline 2：AI Ops（Plan-Execute-Replan）**
   - 数据流图：告警查询 → 规划 → 执行（工具调用）→ 重规划循环 → 分析报告
   - 涉及文件链接
5. **知识索引 Pipeline**
   - 数据流图：文件上传 → 分块 → 向量化 → 入库
6. **Agent 概念索引** — 按 7 个核心概念列出定义和对应代码链接
7. **技术栈说明** — Eino、Milvus、GLM-5.1、MCP 等简述

---

## 层次 2：模块级文档

每个关键模块目录添加 `README.md`：

### `internal/ai/agent/chat_pipeline/README.md`
- RAG Chat Agent 模块概述
- 图编排节点说明（START → RAG → Retriever → Template → ReActAgent → END）
- 关键概念：RAG、ReAct、向量检索、Prompt 模板
- 文件清单和职责

### `internal/ai/agent/plan_execute_replan/README.md`
- Plan-Execute-Replan 模式概述
- 三阶段（规划 → 执行 → 重规划）的角色分工
- Thinking 模型 vs Quick 模型的使用策略
- 文件清单和职责

### `internal/ai/agent/knowledge_index_pipeline/README.md`
- 知识索引 Pipeline 概述
- 文档处理流程（加载 → 分割 → 向量化 → 入库）
- 文件清单和职责

### `internal/ai/tools/README.md`
- Agent 工具集概述
- 每个工具的职责、输入输出
- Tool Use 概念：Tool Schema 定义、MCP 协议
- 文件清单和职责

---

## 层次 3：代码文件注释

### 注释规范

#### 文件头注释格式

```go
/*
=== AI Agent 概念：【概念名称】===

【一句话解释这个概念是什么，解决什么问题】

核心原理：
  - 要点1
  - 要点2

本文件的角色：
  【这个文件在整个 Agent 系统中的位置和职责】

关键数据流：
  输入 → 处理 → 输出

关联文件：
  - xxx.go — 【关系说明】
*/
```

#### 函数/方法注释格式

```go
// functionName 简要说明
//
// 【AI 概念】相关概念的解释
//
// 这里做的事：具体步骤说明
func functionName(...) {
```

#### 代码行注释

关键逻辑行添加中文行注释，解释"为什么这样做"而非"做了什么"。

### 语言规范

- 所有注释使用中文
- 英文专有名词保留英文（RAG、ReAct、SSE、Milvus、embedding、Plan-Execute-Replan、Tool Use 等）
- 架构文档使用中文

### 不改动的部分

- 不修改任何业务逻辑代码
- 不改变代码结构、不重命名、不重构
- 只添加注释和文档

---

## 概念覆盖范围

### 概念 1：RAG（检索增强生成）

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| retriever.go | `internal/ai/agent/chat_pipeline/retriever.go` | Milvus 向量检索，检索参数配置 |
| embedding.go | `internal/ai/agent/chat_pipeline/embedding.go` | 文本向量化，embedding 模型选择 |
| prompt.go | `internal/ai/agent/chat_pipeline/prompt.go` | RAG 文档注入 Prompt 模板的设计 |

**讲解重点：** 用户问题 → 向量化 → 向量检索 → 文档注入 Prompt → LLM 生成

### 概念 2：Agent 编排（Graph / DAG 流水线）

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| orchestration.go | `internal/ai/agent/chat_pipeline/orchestration.go` | RAG Chat 图编排，节点连线 |
| flow.go | `internal/ai/agent/chat_pipeline/flow.go` | BuildChatAgent 入口函数 |
| lambda_func.go | `internal/ai/agent/chat_pipeline/lambda_func.go` | 节点间数据转换函数 |

**讲解重点：** Eino 框架如何用 DAG 图编排 Agent 节点、节点间数据传递、Lambda 转换函数的作用

### 概念 3：ReAct Agent（推理 + 行动）

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| orchestration.go | `internal/ai/agent/chat_pipeline/orchestration.go` | ReAct Agent 节点配置和工具绑定 |
| tools_node.go | `internal/ai/agent/chat_pipeline/tools_node.go` | 工具注册到 Agent |

**讲解重点：** ReAct 模式（Reasoning → Action → Observation 循环）、Agent 如何决定调用哪个工具

### 概念 4：Plan-Execute-Replan

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| plan_execute_replan.go | `internal/ai/agent/plan_execute_replan/plan_execute_replan.go` | 三阶段循环编排 |
| planner.go | `internal/ai/agent/plan_execute_replan/planner.go` | 规划器，生成执行步骤 |
| executor.go | `internal/ai/agent/plan_execute_replan/executor.go` | 执行器，执行步骤并调用工具 |
| replan.go | `internal/ai/agent/plan_execute_replan/replan.go` | 重规划器，根据结果调整计划 |

**讲解重点：** Plan → Execute → Observe → Replan 循环；Thinking 模型（GLM-5.1 Think）用于规划和重规划、Quick 模型用于执行

### 概念 5：Tool Use（工具调用）

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| query_log.go | `internal/ai/tools/query_log.go` | MCP 协议日志查询工具 |
| query_metrics_alerts.go | `internal/ai/tools/query_metrics_alerts.go` | Prometheus 告警查询工具 |
| mysql_crud.go | `internal/ai/tools/mysql_crud.go` | MySQL CRUD 操作工具 |
| get_current_time.go | `internal/ai/tools/get_current_time.go` | 当前时间工具 |
| query_internal_docs.go | `internal/ai/tools/query_internal_docs.go` | 内部文档 RAG 查询工具 |
| tools_node.go | `internal/ai/agent/chat_pipeline/tools_node.go` | 工具注册和 Tool Schema |

**讲解重点：** 如何定义 Agent 工具、Tool Schema 描述（给 LLM 看的函数签名）、MCP 协议的工作方式

### 概念 6：SSE 流式输出

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| sse.go | `internal/logic/sse/sse.go` | Go 后端 SSE 服务实现 |
| chat_v1_chat_stream.go | `internal/controller/chat/chat_v1_chat_stream.go` | SSE Controller 入口 |
| app.js | `OpsPilotFrontend/app.js` | 前端 EventSource 消费 SSE |

**讲解重点：** SSE 协议原理、Go 中如何实现流式响应、前端 EventSource API 使用

### 概念 7：知识索引 Pipeline

**涉及文件：**

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| orchestration.go | `internal/ai/agent/knowledge_index_pipeline/orchestration.go` | 索引图编排 |
| loader.go | `internal/ai/agent/knowledge_index_pipeline/loader.go` | 文件加载器 |
| transformer.go | `internal/ai/agent/knowledge_index_pipeline/transformer.go` | Markdown 分割策略 |
| indexer.go | `internal/ai/agent/knowledge_index_pipeline/indexer.go` | Milvus 写入和 Schema |

**讲解重点：** 文档 → 分块（Markdown Header Splitter）→ 向量化 → 入库（Milvus）的完整流程

---

## 补充注释的辅助文件

以下辅助文件也添加注释，帮助理解基础设施：

| 文件 | 路径 | 注释重点 |
|------|------|----------|
| open_ai.go | `internal/ai/models/open_ai.go` | LLM 模型工厂，GLM-5.1 配置 |
| embedder.go | `internal/ai/embedder/embedder.go` | Embedding 封装 |
| indexer.go | `internal/ai/indexer/indexer.go` | Milvus 索引器 |
| retriever.go | `internal/ai/retriever/retriever.go` | Milvus 检索器封装 |
| mem.go | `utility/mem/mem.go` | 对话记忆管理（滑动窗口） |
| client.go | `utility/client/client.go` | Milvus 客户端初始化 |

---

## 文件完整清单

**新建文件（2 个）：**
1. `docs/ai-agent-architecture-guide.md` — 顶层架构文档
2. 各模块 `README.md`（4 个模块目录）

**添加注释的文件（约 20 个）：**

`internal/ai/agent/chat_pipeline/` 目录（5 个）：
- orchestration.go, flow.go, prompt.go, retriever.go, embedding.go, lambda_func.go, tools_node.go, types.go

`internal/ai/agent/plan_execute_replan/` 目录（4 个）：
- plan_execute_replan.go, planner.go, executor.go, replan.go

`internal/ai/agent/knowledge_index_pipeline/` 目录（4 个）：
- orchestration.go, loader.go, transformer.go, indexer.go

`internal/ai/tools/` 目录（5 个）：
- query_log.go, query_metrics_alerts.go, mysql_crud.go, get_current_time.go, query_internal_docs.go

`internal/logic/sse/` 目录（1 个）：
- sse.go

`internal/controller/chat/` 目录（1 个）：
- chat_v1_chat_stream.go

`internal/ai/models/` 目录（1 个）：
- open_ai.go

辅助文件（5 个）：
- internal/ai/embedder/embedder.go
- internal/ai/indexer/indexer.go
- internal/ai/retriever/retriever.go
- utility/mem/mem.go
- utility/client/client.go

前端文件（1 个）：
- OpsPilotFrontend/app.js（仅 SSE 相关部分）

---

## 产出物交付清单

| # | 产出物 | 类型 |
|---|--------|------|
| 1 | `docs/ai-agent-architecture-guide.md` | 新建 |
| 2 | `internal/ai/agent/chat_pipeline/README.md` | 新建 |
| 3 | `internal/ai/agent/plan_execute_replan/README.md` | 新建 |
| 4 | `internal/ai/agent/knowledge_index_pipeline/README.md` | 新建 |
| 5 | `internal/ai/tools/README.md` | 新建 |
| 6 | 约 20 个代码文件的中文注释 | 修改（仅添加注释） |
