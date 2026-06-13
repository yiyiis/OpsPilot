/*
=== AI Agent 概念：ReAct Agent（推理 + 行动）===

ReAct（Reasoning + Acting）是让 LLM 自主决策的核心模式。
每轮循环：推理当前情况 → 决定是否调用工具 → 执行工具 → 观察结果 → 继续推理或输出回答。

核心原理：
  - LLM 看到可用工具的描述（Tool Schema），自主决定调用哪个工具
  - 每次工具调用后，结果作为"观察"反馈给 LLM
  - LLM 可以进行多轮推理-行动循环（最多 MaxStep 次）
  - 如果 LLM 认为已有足够信息，直接输出最终回答（不调用工具）

本文件的角色：
  构建 ReAct Agent 实例——配置 LLM 模型、绑定可用工具、包装为 DAG 图节点。
  这是 Chat Pipeline 的"大脑"节点。

关联文件：
  - model.go — 提供 LLM 模型实例（GLM-5.1 Quick）
  - tools/ — 提供各工具的实现
  - tools_node.go — DuckDuckGo 搜索工具（当前未启用）
*/
package chat_pipeline

import (
	"OpsPilot/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

// newReactAgentLambda 构建 ReAct Agent 并包装为 Lambda 节点
//
// 【AI 概念】ReAct Agent 的核心配置：
//   - MaxStep: 最大推理-行动循环次数（25次），防止无限循环
//   - ToolCallingModel: 用于推理的 LLM 模型
//   - Tools: Agent 可调用的工具列表
//   - ToolReturnDirectly: 哪些工具的结果可以直接作为最终回答（空=都需要 LLM 再处理）
//
// 【数据流】
//   输入：ChatTemplate 组装好的 Prompt（系统提示 + RAG 文档 + 历史 + 用户问题）
//   处理：LLM 推理 → 可能调用工具 → 观察结果 → 继续推理...
//   输出：最终回答（*schema.Message）
func newReactAgentLambda(ctx context.Context) (lba *compose.Lambda, err error) {
	// 配置 ReAct Agent
	config := &react.AgentConfig{
		MaxStep:            25,                       // 最多 25 轮推理-行动循环
		ToolReturnDirectly: map[string]struct{}{},     // 空 map = 所有工具结果都需要 LLM 进一步处理
	}

	// 获取 LLM 模型（GLM-5.1 Quick，快速响应模式）
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = chatModelIns11

	// === 注册可用工具 ===
	// LLM 会根据工具的 Schema 描述自主决定调用哪个工具

	// 注意：DuckDuckGo 搜索工具当前被注释掉，未启用
	// searchTool, err := newSearchTool(ctx)

	// 工具1：MCP 工具集 — 通过 MCP 协议连接多个外部服务（日志查询、联网搜索、网页读取等）
	mcpTool, err := tools.GetAllMcpTools()
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = mcpTool // MCP 返回的是工具列表（可能有多个工具）

	// 工具2：Prometheus 告警查询
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewPrometheusAlertsQueryTool())
	// 工具3：MySQL 数据库操作
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewMysqlCrudTool())
	// 工具4：获取当前时间
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewGetCurrentTimeTool())
	// 工具5：内部文档 RAG 检索（工具内部也使用了 Milvus 检索）
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewQueryInternalDocsTool())

	// 创建 ReAct Agent 实例
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	// 将 Agent 包装为 Lambda 节点，使其可以嵌入 DAG 图
	// AnyLambda 接受 Generate（同步）和 Stream（流式）两个方法
	// 这样图既能同步执行也能流式执行
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
