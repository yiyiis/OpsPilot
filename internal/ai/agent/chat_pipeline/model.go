/*
=== AI Agent 概念：LLM 模型实例化 ===

本文件为 Chat Pipeline 创建 LLM 模型实例。
使用 GLM-5.1 Quick 模型（快速响应模式），适合对话场景。

本文件的角色：
  作为 chat_pipeline 包内部的模型创建入口。
  委托给 internal/ai/models 包的统一工厂方法。

关联文件：
  - internal/ai/models/open_ai.go — 模型工厂的统一入口
  - flow.go — 调用 newChatModel 获取模型实例
*/
package chat_pipeline

import (
	"OpsPilot/internal/ai/models"
	"context"

	"github.com/cloudwego/eino/components/model"
)

// newChatModel 创建 Chat Agent 使用的 LLM 模型
//
// 【AI 概念】Chat Agent 为什么用 Quick 模型？
// 对话场景中，用户期望快速回复。Quick 模型延迟更低，
// 直接输出回答而不展示推理过程，适合实时对话。
//
// 对比：Plan-Execute-Replan 的 Planner 和 Replanner 使用 Think 模型
// （需要深度推理，延迟可接受）
func newChatModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	cm, err = models.OpenAIForDeepSeekV3Quick(ctx)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
