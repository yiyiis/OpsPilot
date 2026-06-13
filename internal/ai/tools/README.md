# Agent Tools 工具集模块

## 概述

本模块定义了 OpsPilot 可调用的 **Agent 工具（Tools）**。每个工具是一个独立的功能单元，LLM 可以在推理过程中自主决定是否调用。

## 核心概念

### Tool Use（工具调用）

LLM 本身只能生成文本，无法访问外部数据或执行操作。Tool Use 让 LLM 能够：

1. **阅读工具描述**（Tool Schema）——了解有哪些工具可用、每个工具需要什么参数
2. **决定是否调用**——根据当前推理判断是否需要使用某个工具
3. **生成调用参数**——按 Schema 要求生成 JSON 格式的参数
4. **接收返回结果**——工具执行后，结果会作为"观察"反馈给 LLM

### MCP（Model Context Protocol）

MCP 是一种标准化的工具集成协议。通过 MCP，Agent 可以连接到外部工具服务器，动态获取可用工具列表，而无需在代码中硬编码每个工具的实现。

本项目中，日志查询工具通过 MCP 协议连接到日志服务。

## 工具清单

| 工具 | 文件 | 功能 | 底层技术 | 状态 |
|------|------|------|----------|------|
| `query_log` | `query_log.go` | 查询日志 | MCP (SSE 协议) | ✅ 启用 |
| `query_prometheus_alerts` | `query_metrics_alerts.go` | 查询 Prometheus 告警 | HTTP API | ⚠️ HTTP 调用被注释，返回空 |
| `mysql_crud` | `mysql_crud.go` | MySQL 数据库操作 | GORM | ✅ 启用 |
| `get_current_time` | `get_current_time.go` | 获取当前时间 | 标准库 | ✅ 启用 |
| `query_internal_docs` | `query_internal_docs.go` | 内部文档 RAG 检索 | Milvus | ✅ 启用 |

## 工具定义模式

本项目使用 Eino 框架的 `utils.InferOptionableTool` 来定义工具，它会：

1. 从 Go 结构体的 `jsonschema` 标签自动生成 Tool Schema（给 LLM 看的函数签名）
2. 将 Go 函数包装为 Eino 的 `tool.InvokableTool` 接口
3. LLM 看到的工具描述来自 `jsonschema` 标签中的 `description` 字段

```go
// 示例：工具输入结构体
type ToolInput struct {
    Query string `json:"query" jsonschema:"description=要查询的内容"`
}

// 示例：工具定义
tool.NewTool(
    "tool_name",                    // 工具名（LLM 看到的）
    "工具描述",                      // 功能描述（LLM 看到的）
    utils.InferOptionableTool(func...) // 处理函数
)
```

## 推荐阅读顺序

1. `get_current_time.go` — 最简单的工具，理解 Tool 定义模式
2. `query_internal_docs.go` — 工具内部使用 RAG，展示工具组合
3. `query_log.go` — MCP 协议集成，理解动态工具获取
4. `query_metrics_alerts.go` — 较复杂的工具，含数据结构定义和去重逻辑
5. `mysql_crud.go` — 数据库操作工具，含交互式确认
