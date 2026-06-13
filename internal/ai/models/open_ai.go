/*
=== AI Agent 概念：LLM 模型工厂 ===

本文件是所有 LLM 模型的创建入口。通过工厂模式，统一管理不同模型的配置和实例化。
项目使用智谱 GLM-5.1 模型，各司其职。

核心原理：
  - 通过 OpenAI 兼容接口调用 GLM-5.1 模型（智谱 Coding 套餐）
  - 两种模型各有适用场景：
    - Think 模型：展示推理过程，适合需要深度推理的任务（规划、重规划）
    - Quick 模型：快速响应，适合需要快速执行的任务（对话、工具调用）

模型选择策略：

  | 角色           | 模型                | 原因                     |
  |----------------|---------------------|--------------------------|
  | RAG Chat Agent | GLM-5.1 Quick       | 对话场景需要快速响应       |
  | Planner        | GLM-5.1 Think       | 规划需要深度推理           |
  | Executor       | GLM-5.1 Quick       | 执行步骤需要快速响应       |
  | Replanner      | GLM-5.1 Think       | 重规划需要深度推理         |

关联文件：
  - chat_pipeline/model.go — Chat Agent 使用 Quick 模型
  - plan_execute_replan/planner.go — Planner 使用 Think 模型
  - plan_execute_replan/executor.go — Executor 使用 Quick 模型
  - plan_execute_replan/replan.go — Replanner 使用 Think 模型
*/
package models

import (
	"context"

	"OpsPilot/utility/config"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// OpenAIForDeepSeekV31Think 创建 GLM-5.1 Think 模型（思考模式）
//
// 【AI 概念】Thinking Model（思考模型）
// Think 模型在生成回答前会先输出一段推理过程（<think/> 块），
// 然后再给出最终结论。这种"慢思考"模式适合需要深度推理的任务：
//   - 分析复杂问题
//   - 制定多步计划
//   - 评估执行结果的充分性
//
// 配置来源：config.yaml 中的 glm_think_chat_model 配置项
func OpenAIForDeepSeekV31Think(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	// 从 Viper 配置中读取模型参数
	configObj := &openai.ChatModelConfig{
		Model:   config.App.GLMThinkChat.Model,
		APIKey:  config.App.GLMThinkChat.APIKey,
		BaseURL: config.App.GLMThinkChat.BaseURL,
	}

	// 使用 Eino 的 OpenAI 兼容适配器创建模型
	// GLM-5.1 通过智谱 Coding 套餐提供 OpenAI 兼容接口
	cm, err = openai.NewChatModel(ctx, configObj)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// OpenAIForDeepSeekV3Quick 创建 GLM-5.1 Quick 模型（快速模式）
//
// 【AI 概念】Quick Model（快速模型）
// Quick 模型直接输出回答，不展示推理过程。延迟更低，适合：
//   - 对话场景（用户期望快速回复）
//   - 工具调用（执行具体步骤，不需要深度推理）
//
// 两个函数的结构完全相同，只是配置前缀不同：
//   - Think → glm_think_chat_model
//   - Quick → glm_quick_chat_model
//
// 配置来源：config.yaml 中的 glm_quick_chat_model 配置项
func OpenAIForDeepSeekV3Quick(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	configObj := &openai.ChatModelConfig{
		Model:   config.App.GLMQuickChat.Model,
		APIKey:  config.App.GLMQuickChat.APIKey,
		BaseURL: config.App.GLMQuickChat.BaseURL,
	}
	cm, err = openai.NewChatModel(ctx, configObj)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
