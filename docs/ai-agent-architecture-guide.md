# OpsPilot AI Agent 架构指南

> 本文档从 AI Agent 技术的视角，系统梳理 OpsPilot 的架构设计。
> 面向有一定 LLM 基础、想学习复杂 Agent 模式的开发者。

---

## 一、项目概述

OpsPilot 是一个 **AI 驱动的智能 OnCall 助手**，帮助运维人员：

- 通过对话查询内部运维知识库（RAG）
- 自动分析服务告警，生成诊断报告（Plan-Execute-Replan）
- 查询日志（MCP）、监控告警（Prometheus）、数据库（MySQL）等外部系统

**技术栈：**

| 组件 | 技术选型 | 作用 |
|------|----------|------|
| Web 框架 | Gin | HTTP 路由、中间件、配置管理 |
| Agent 框架 | CloudWeGo Eino | 图编排、ReAct Agent、工具调用 |
| LLM | GLM-5.1（智谱） | 对话生成、规划推理 |
| Embedding | Google Gemini embedding-2 | 文本向量化 |
| 向量数据库 | Milvus v2.5 | 知识存储与检索 |
| 工具协议 | MCP (Model Context Protocol) | 外部工具集成（日志查询） |
| 前端 | 原生 JS + SSE | Gemini 风格聊天界面 |

---

## 二、系统架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        用户浏览器 (:8080)                            │
│                    Gemini 风格聊天 + AI Ops 面板                      │
└───────────────────────────┬─────────────────────────────────────────┘
                            │ SSE / HTTP
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Gin HTTP 服务 (:6872)                            │
│  ┌──────────┐  ┌───────────────┐  ┌───────────┐  ┌──────────────┐ │
│  │ /chat    │  │ /chat_stream  │  │ /upload   │  │ /ai_ops      │ │
│  │ (同步)   │  │ (SSE 流式)    │  │ (文件上传) │  │ (告警分析)    │ │
│  └────┬─────┘  └──────┬────────┘  └─────┬─────┘  └──────┬───────┘ │
└───────┼───────────────┼─────────────────┼───────────────┼──────────┘
        │               │                 │               │
        ▼               ▼                 ▼               ▼
┌───────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐
│ RAG Chat      │ │ RAG Chat     │ │ Knowledge    │ │ Plan-Execute-Replan  │
│ Agent (同步)  │ │ Agent (流式)  │ │ Index        │ │ Agent                │
│               │ │              │ │ Pipeline     │ │                      │
│ Eino Graph:   │ │ Eino Graph:  │ │              │ │ Eino ADK:            │
│ ┌───────────┐ │ │ ┌──────────┐ │ │ Eino Graph:  │ │ Planner (Think模型)  │
│ │ RAG检索   │ │ │ │ RAG检索  │ │ │              │ │   ↓                  │
│ │  ↓        │ │ │ │  ↓       │ │ │ FileLoader   │ │ Executor (Quick模型) │
│ │ Prompt组装│ │ │ │ Prompt   │ │ │   ↓          │ │   ↓                  │
│ │  ↓        │ │ │ │  ↓       │ │ │ MdSplitter   │ │ Replanner (Think模型)│
│ │ ReAct     │ │ │ │ ReAct    │ │ │   ↓          │ │                      │
│ │ Agent     │ │ │ │ Agent    │ │ │ MilvusWriter │ │ 工具: 日志/告警/SQL/  │
│ └───────────┘ │ │ └──────────┘ │ │              │ │ 时间/内部文档         │
└───────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘
        │               │                                      │
        └───────────────┼──────────────────────────────────────┘
                        ▼
        ┌───────────────────────────────────┐
        │           共享基础设施              │
        │  ┌──────────┐  ┌───────────────┐  │
        │  │ Milvus   │  │ 对话记忆      │  │
        │  │ 向量数据库│  │ (内存Map)     │  │
        │  └──────────┘  └───────────────┘  │
        │  ┌──────────┐  ┌───────────────┐  │
        │  │ LLM 工厂  │  │ Embedding     │  │
        │  │ (GLM-5.1) │  │ (Gemini)      │  │
        │  └──────────┘  └───────────────┘  │
        └───────────────────────────────────┘
                        │
        ┌───────────────┼───────────────────┐
        ▼               ▼                   ▼
  ┌──────────┐   ┌──────────┐        ┌──────────┐
  │ MCP 服务 │   │Prometheus│        │ MySQL    │
  │ (日志查询)│   │ (告警)   │        │ (数据库) │
  └──────────┘   └──────────┘        └──────────┘
```

---

## 三、核心 Pipeline 1：RAG Chat

**概念：** RAG（Retrieval-Augmented Generation，检索增强生成）让 LLM 基于外部知识库回答问题，而非仅依赖训练数据。

**数据流：**

```
用户问题 "服务 panic 怎么排查？"
        │
        ▼
┌─ InputToRag ──────────────────────────────────────────────────┐
│ 提取用户问题文本（Lambda 转换函数）                               │
│ 输入: UserMessage{Query: "服务 panic 怎么排查？"}               │
│ 输出: "服务 panic 怎么排查？"                                    │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌─ MilvusRetriever ────────────────────────────────────────────┐
│ 【核心 RAG 步骤】向量检索                                       │
│ 1. 用户问题 → Embedding 向量化                                 │
│ 2. 在 Milvus 中搜索语义最相近的文档片段（TopK）                   │
│ 输出: []schema.Document（相关文档片段列表）                       │
│        内容如: "## 服务 panic 处理\n1. 查看 stderr 日志..."      │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌───────────── InputToChat (并行分支) ────────────────────────────┐
│ 同时，另一路提取对话上下文                                        │
│ 输入: UserMessage{Query, History, ID}                          │
│ 输出: map{"content": 问题, "history": 历史消息, "date": 时间}    │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼         ← 两路数据在 ChatTemplate 汇聚
┌─ ChatTemplate ───────────────────────────────────────────────┐
│ 组装最终 Prompt：                                              │
│ ┌─────────────────────────────────────────────────────────┐  │
│ │ System: 你是一个智能 OnCall 助手...                        │  │
│ │ 参考文档: {documents}  ← 来自 MilvusRetriever            │  │
│ │ 日期: {date}           ← 来自 InputToChat                │  │
│ │ History: {history}     ← 来自 InputToChat                │  │
│ │ User: {content}        ← 来自 InputToChat                │  │
│ └─────────────────────────────────────────────────────────┘  │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌─ ReActAgent ─────────────────────────────────────────────────┐
│ 【核心 Agent 步骤】推理 + 行动循环                               │
│                                                               │
│ LLM 推理: "用户问 panic 排查，参考文档提到了日志查看..."          │
│     ↓                                                        │
│ 选择工具: 需要查询日志？ → 调用 query_log 工具                    │
│     ↓                                                        │
│ 观察结果: 获取到日志内容                                         │
│     ↓                                                        │
│ 继续推理: 基于日志和文档，生成最终回答                              │
│                                                               │
│ 可用工具:                                                      │
│  • query_log (MCP 日志查询)                                    │
│  • query_prometheus_alerts (告警查询)                           │
│  • mysql_crud (数据库操作)                                      │
│  • get_current_time (获取时间)                                  │
│  • query_internal_docs (内部文档 RAG)                           │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
         返回回答给用户
```

**关键代码文件：** [`internal/ai/agent/chat_pipeline/`](internal/ai/agent/chat_pipeline/)

---

## 四、核心 Pipeline 2：AI Ops（Plan-Execute-Replan）

**概念：** Plan-Execute-Replan 是一种高级 Agent 模式。先让 LLM 制定执行计划，然后逐步执行，根据执行结果动态调整计划。适合复杂的多步骤任务。

**模型分工策略：**
- **GLM-5.1（智谱，思考模型）** → 用于规划和重规划（需要深度推理）
- **GLM-5.1（智谱，快速模型）** → 用于执行（需要快速响应）

**数据流：**

```
用户告警分析请求
        │
        ▼
┌─ Planner（规划器）─────────────────────────────────────────────┐
│ 模型: GLM-5.1（智谱，思考模型）                                    │
│                                                                │
│ 输入: "分析当前服务告警状态"                                      │
│                                                                │
│ 输出: 执行计划                                                   │
│   Step 1: 查询 Prometheus 获取当前活跃告警                       │
│   Step 2: 查询日志获取相关错误信息                                 │
│   Step 3: 查询内部文档获取处理手册                                │
│   Step 4: 综合分析生成报告                                       │
└───────────────┬────────────────────────────────────────────────┘
                │
                ▼
┌─ Executor（执行器）─────────────────────────────────────────────┐
│ 模型: GLM-5.1（智谱，快速模型）                                    │
│                                                                │
│ 逐步执行计划中的步骤，每步可调用工具：                              │
│                                                                │
│   Step 1 → 调用 query_prometheus_alerts                        │
│          ← 返回: 3 个活跃告警                                   │
│                                                                │
│   Step 2 → 调用 query_log                                      │
│          ← 返回: panic 错误日志                                 │
│                                                                │
│   Step 3 → 调用 query_internal_docs                            │
│          ← 返回: 告警处理手册相关章节                             │
│                                                                │
│ 可用工具: query_log / query_prometheus_alerts /                 │
│         query_internal_docs / get_current_time                  │
└───────────────┬────────────────────────────────────────────────┘
                │
                ▼
┌─ Replanner（重规划器）──────────────────────────────────────────┐
│ 模型: GLM-5.1（智谱，思考模型）                                    │
│                                                                │
│ 分析执行结果，决定：                                              │
│   ✓ 计划已完成 → 输出最终分析报告                                 │
│   ✗ 需要更多信息 → 修改计划，回到 Executor 继续执行                │
│                                                                │
│ 最多循环 20 次（MaxIterations: 20）                               │
└────────────────────────────────────────────────────────────────┘
```

**关键代码文件：** [`internal/ai/agent/plan_execute_replan/`](internal/ai/agent/plan_execute_replan/)

---

## 五、知识索引 Pipeline

**概念：** 将文档转化为向量并存入向量数据库，是 RAG 的前置步骤。

**数据流：**

```
文件上传（.md / .txt）
        │
        ▼
┌─ FileLoader ──────────────────────────────────────────────────┐
│ 读取文件内容，转为 Eino document.Source 对象                     │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌─ MarkdownSplitter ────────────────────────────────────────────┐
│ 按 Markdown 标题（# H1）分割文档为多个片段                        │
│ 每个片段分配唯一 UUID                                           │
│                                                                │
│ 例如: 告警处理手册.md →                                          │
│   片段1: "## 服务宕机处理\n内容..."                               │
│   片段2: "## 高失败率处理\n内容..."                               │
│   片段3: "## 对账差异处理\n内容..."                               │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌─ MilvusIndexer ───────────────────────────────────────────────┐
│ 对每个片段:                                                     │
│ 1. 文本 → Embedding 向量化（Gemini embedding-2）                  │
│ 2. 写入 Milvus 向量数据库                                        │
│                                                                │
│ 存储结构:                                                       │
│   id (主键) | vector (向量, dim=3072) | content | metadata      │
└───────────────────────────────────────────────────────────────┘
```

**关键代码文件：** [`internal/ai/agent/knowledge_index_pipeline/`](internal/ai/agent/knowledge_index_pipeline/)

---

## 六、Agent 核心概念索引

| # | 概念 | 一句话解释 | 核心代码文件 |
|---|------|-----------|-------------|
| 1 | **RAG（检索增强生成）** | 让 LLM 基于外部知识库回答问题 | [chat_pipeline/retriever.go](internal/ai/agent/chat_pipeline/retriever.go)、[chat_pipeline/prompt.go](internal/ai/agent/chat_pipeline/prompt.go) |
| 2 | **Agent 编排（Graph/DAG）** | 用有向无环图编排多个处理步骤 | [chat_pipeline/orchestration.go](internal/ai/agent/chat_pipeline/orchestration.go) |
| 3 | **ReAct Agent** | 推理→行动→观察循环，让 LLM 自主选择工具 | [chat_pipeline/flow.go](internal/ai/agent/chat_pipeline/flow.go) |
| 4 | **Plan-Execute-Replan** | 先规划→再执行→根据结果重规划 | [plan_execute_replan/](internal/ai/agent/plan_execute_replan/) |
| 5 | **Tool Use（工具调用）** | 让 LLM 调用外部函数获取实时数据 | [tools/](internal/ai/tools/) |
| 6 | **SSE 流式输出** | 服务端向客户端实时推送数据 | [sse.go](internal/logic/sse/sse.go) |
| 7 | **知识索引** | 文档分块→向量化→入库 | [knowledge_index_pipeline/](internal/ai/agent/knowledge_index_pipeline/) |

---

## 七、推荐阅读顺序

1. **先理解工具（最简单）：** `internal/ai/tools/get_current_time.go` — 最简单的 Tool 示例
2. **理解 RAG 检索：** `internal/ai/agent/chat_pipeline/retriever.go` → `prompt.go`
3. **理解图编排：** `internal/ai/agent/chat_pipeline/orchestration.go` — DAG 是如何构建的
4. **理解 ReAct Agent：** `internal/ai/agent/chat_pipeline/flow.go` — 工具是如何绑定到 Agent 的
5. **理解 Plan-Execute-Replan：** `internal/ai/agent/plan_execute_replan/` — 4 个文件按顺序读
6. **理解 SSE 流式：** `internal/logic/sse/sse.go` → `chat_v1_chat_stream.go`
7. **理解知识索引：** `internal/ai/agent/knowledge_index_pipeline/`
