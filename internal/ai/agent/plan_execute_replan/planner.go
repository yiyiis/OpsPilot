/*
=== AI Agent 概念：规划器（Planner）===

规划器是 Plan-Execute-Replan 模式的第一阶段。
它接收用户的问题，分析后制定一个分步执行计划。

核心原理：
  - 使用"思考模型"（Thinking Model）进行深度推理
  - 输出结构化的执行步骤列表
  - 每个步骤描述需要做什么、调用什么工具

本文件的角色：
  创建 Planner 实例——配置使用 GLM-5.1 Think 模型。

模型选择原因：
  GLM-5.1 Think 是"思考模型"，会先展示推理过程再给出结论。
  规划需要深度推理（分析问题 → 拆解步骤 → 预判依赖），所以用 Think 模型。
  相比之下，执行步骤用的是 Quick 模型（快速但不展示推理过程）。

关联文件：
  - plan_execute_replan.go — 顶层编排，调用 NewPlanner
  - internal/ai/models/open_ai.go — 提供 OpenAIForDeepSeekV31Think（GLM-5.1 Think）模型
*/
package plan_execute_replan

import (
	"OpsPilot/internal/ai/models"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// NewPlanner 创建规划器 Agent
//
// 【AI 概念】为什么规划器用 Think 模型？
// 规划需要"慢思考"——分析问题、拆解步骤、预判可能的困难。
// Think 模型会先输出一段推理过程（<think/> 块），再给出最终计划。
// 这种深度推理能力是 Quick 模型不具备的。
//
// 返回：adk.Agent — 可被 planexecute 框架使用的规划器实例
func NewPlanner(ctx context.Context) (adk.Agent, error) {
	// 使用 GLM-5.1 Think 模型（思考模式）
	planModel, err := models.OpenAIForDeepSeekV31Think(ctx)
	if err != nil {
		return nil, err
	}

	// 使用 Eino ADK 的预构建 Planner
	// Planner 会将 LLM 的输出解析为结构化的执行计划
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: planModel, // LLM 模型，用于生成执行计划
	})
}
