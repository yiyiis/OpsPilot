/*
=== AI Agent 概念：重规划器（Replanner）===

重规划器是 Plan-Execute-Replan 模式的第三阶段。
它评估执行器的执行结果，决定是完成任务还是调整计划继续执行。

核心原理：
  - 接收：原计划 + 已执行的步骤 + 执行结果
  - 推理：分析执行结果是否足以回答原始问题
  - 决策：
       ✓ 信息充足 → 输出最终分析报告（循环结束）
       ✗ 还需要更多信息 → 修改计划，补充新步骤（回到 Executor）

本文件的角色：
  创建 Replanner 实例——配置使用 GLM-5.1 Think 模型。

模型选择原因：
  重规划需要深度推理——评估"当前结果是否足够"需要理解问题的全貌，
  判断"还缺什么信息"需要分析能力。这与 Planner 的需求一致，
  所以使用同款 Think 模型。

关联文件：
  - plan_execute_replan.go — 顶层编排，调用 NewRePlanAgent
  - planner.go — 使用同款 Think 模型的规划器
  - executor.go — 使用 Quick 模型的执行器（对比模型选择策略）
*/
package plan_execute_replan

import (
	"OpsPilot/internal/ai/models"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// NewRePlanAgent 创建重规划器 Agent
//
// 【AI 概念】重规划的触发条件
// 重规划器在每次执行器完成一轮执行后被调用。它会判断：
//
//   1. 计划中的所有步骤是否都已执行完成？
//   2. 执行结果是否足以回答原始问题？
//   3. 是否发现了新的需要调查的问题？
//
// 根据判断结果，重规划器会：
//   - 输出最终回答（如果信息充足）
//   - 修改原计划（如果需要更多信息）
//   - 添加新步骤（如果发现了新问题）
//
// 注意：重规划器不需要工具（它只做推理和判断，不直接获取数据）
// 所以配置中只有 ChatModel，没有 ToolsConfig
func NewRePlanAgent(ctx context.Context) (adk.Agent, error) {
	// 使用与 Planner 相同的 Think 模型
	// 重规划需要深度推理来评估执行结果的充分性
	model, err := models.OpenAIForDeepSeekV31Think(ctx)
	if err != nil {
		return nil, err
	}

	// 创建重规划器
	// ReplannerConfig 只需要 ChatModel，不需要工具
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
}
