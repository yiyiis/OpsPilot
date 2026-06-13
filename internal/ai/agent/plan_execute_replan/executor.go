/*
=== AI Agent 概念：执行器（Executor）===

执行器是 Plan-Execute-Replan 模式的第二阶段。
它按照规划器制定的计划，逐步执行每个步骤，每步可以调用工具获取数据。

核心原理：
  - 接收规划器生成的步骤列表
  - 对每个步骤：LLM 推理 → 选择工具 → 调用 → 观察结果
  - 执行完所有步骤后，将结果传递给重规划器评估

本文件的角色：
  创建 Executor 实例——绑定 4 个工具，配置使用 GLM-5.1 Quick 模型。

模型选择原因：
  执行器需要快速响应——每个步骤的执行不需要深度推理，
  只需要根据步骤描述选择合适的工具并调用。Quick 模型更适合这种场景。

可用工具：
  - query_log (MCP) — 查询日志
  - query_prometheus_alerts — 查询 Prometheus 告警
  - query_internal_docs — RAG 检索内部文档
  - get_current_time — 获取当前时间

关联文件：
  - plan_execute_replan.go — 顶层编排，调用 NewExecutor
  - internal/ai/tools/ — 各工具的实现
  - internal/ai/models/open_ai.go — 提供 OpenAIForDeepSeekV3Quick（GLM-5.1 Quick）模型
*/
package plan_execute_replan

import (
	"OpsPilot/internal/ai/models"
	"OpsPilot/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
)

// NewExecutor 创建执行器 Agent
//
// 【AI 概念】执行器的工具绑定
// 执行器不包含 Planner 和 Replanner 的工具（它们不需要工具）。
// 只有执行器需要工具——它通过工具获取外部数据来完成执行步骤。
//
// 【AI 概念】为什么执行器用 Quick 模型？
// 执行器需要快速响应：收到一个步骤 → 选择工具 → 调用 → 返回结果。
// 不需要深度推理，Quick 模型的延迟更低。
//
// MaxIterations: 999999 — 执行器内部循环次数设得很大，
// 确保执行器不会因为内部迭代限制而提前终止。
// 真正的循环控制由外层 planexecute 的 MaxIterations: 20 管理。
func NewExecutor(ctx context.Context) (adk.Agent, error) {
	// === 组装工具列表 ===

	// 工具1：MCP 工具集（返回工具列表，包含日志查询、联网搜索、网页读取等）
	mcpTool, err := tools.GetAllMcpTools()
	if err != nil {
		return nil, err
	}
	toolList := mcpTool

	// 工具2：Prometheus 告警查询
	toolList = append(toolList, tools.NewPrometheusAlertsQueryTool())

	// 工具3：内部文档 RAG 检索
	toolList = append(toolList, tools.NewQueryInternalDocsTool())

	// 工具4：获取当前时间
	toolList = append(toolList, tools.NewGetCurrentTimeTool())

	// 获取 Quick 模型（快速响应，不展示推理过程）
	execModel, err := models.OpenAIForDeepSeekV3Quick(ctx)
	if err != nil {
		return nil, err
	}

	// 创建执行器
	// 执行器会按计划逐步执行，每步可调用上述工具
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: execModel, // LLM 模型，用于理解步骤并选择工具
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList, // 可调用的工具列表
			},
		},
		MaxIterations: 999999, // 内部循环上限设得很大
	})
}
