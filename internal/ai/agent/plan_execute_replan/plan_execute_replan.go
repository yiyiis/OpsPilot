/*
=== AI Agent 概念：Plan-Execute-Replan 编排 ===

Plan-Execute-Replan 是一种高级 Agent 模式，用于处理复杂的多步骤任务。
与 ReAct 的"边想边做"不同，它先制定完整计划，再逐步执行，根据结果动态调整。

核心原理：
  - Plan（规划）：LLM 分析问题，制定分步执行计划
  - Execute（执行）：按计划逐步执行，每步可调用工具
  - Replan（重规划）：评估执行结果，决定是继续、调整计划还是完成
  - 循环直到任务完成或达到最大迭代次数

本文件的角色：
  顶层编排器——组装 Planner + Executor + Replanner 三个子 Agent，
  运行事件流，收集最终结果和详细日志。

关键数据流：
  用户查询 → Planner（生成计划）→ Executor（按计划执行+调用工具）
  → Replanner（评估结果）→ 循环或输出最终回答

模型选择策略：
  - Planner + Replanner → GLM-5.1 Think（需要深度推理）
  - Executor → GLM-5.1 Quick（需要快速响应）

关联文件：
  - planner.go — 规划器实现
  - executor.go — 执行器实现（绑定工具）
  - replan.go — 重规划器实现
*/
package plan_execute_replan

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// BuildPlanAgent 构建 Plan-Execute-Replan Agent 并执行用户查询
//
// 【AI 概念】Plan-Execute-Replan 的完整循环
//
// 三阶段工作流：
//
//   Planner（Think 模型）
//     "分析当前告警状态" → 制定计划:
//       Step1: 查询 Prometheus 告警
//       Step2: 查询日志
//       Step3: 查询内部文档
//       Step4: 综合分析
//          ↓
//   Executor（Quick 模型）
//     执行 Step1 → 调用 query_prometheus_alerts → 获取告警列表
//     执行 Step2 → 调用 query_log → 获取错误日志
//     ...
//          ↓
//   Replanner（Think 模型）
//     评估执行结果 → 计划已完成? → 输出最终报告
//                         未完成? → 调整计划 → 回到 Executor
//
// 参数：
//   - ctx: 上下文
//   - query: 用户的告警分析请求（如"分析当前服务告警状态"）
//
// 返回值：
//   - string: 最终分析报告
//   - []string: 详细的执行日志（每个步骤的输出）
//   - error: 错误信息
func BuildPlanAgent(ctx context.Context, query string) (string, []string, error) {
	// 创建三个子 Agent
	planAgent, err := NewPlanner(ctx) // 规划器：制定执行计划
	if err != nil {
		return "", []string{}, err
	}
	executeAgent, err := NewExecutor(ctx) // 执行器：执行计划中的步骤
	if err != nil {
		return "", []string{}, err
	}
	replanAgent, err := NewRePlanAgent(ctx) // 重规划器：评估并调整计划
	if err != nil {
		return "", []string{}, err
	}

	// 组装 Plan-Execute-Replan Agent
	// MaxIterations: 20 — 最多循环 20 次（规划→执行→重规划 算一轮）
	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 20,
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("build PlanExecuteAgent Error: %v", err)
	}

	// 创建 ADK Runner 并启动查询
	// Runner 是 Eino ADK 的执行引擎，管理 Agent 的运行生命周期
	r := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: planExecuteAgent,
	})

	// 通过迭代器消费事件流
	// 【AI 概念】事件流模式
	// Agent 的执行过程会产生一系列事件（规划、执行、观察等），
	// 通过迭代器逐个消费，可以实时跟踪进度
	iter := r.Query(ctx, query)
	var lastMessage adk.Message
	var detail []string
	for {
		event, ok := iter.Next()
		if !ok {
			break // 事件流结束
		}
		fmt.Println("------------- Event -------------")
		prints.Event(event) // 打印事件详情（用于调试）
		if event.Output != nil {
			// 提取事件中的消息内容
			lastMessage, _, err = adk.GetMessage(event)
			detail = append(detail, lastMessage.String())
		}
	}

	// 如果没有产出任何消息，返回错误
	if lastMessage == nil {
		return "", []string{}, fmt.Errorf("get lastMessage Error")
	}

	// 返回最终回答和详细执行日志
	return lastMessage.Content, detail, nil
}
