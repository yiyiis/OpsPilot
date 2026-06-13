/*
=== AI Agent 概念：RAG Prompt 工程 ===

在 RAG 系统中，Prompt 模板的设计至关重要。本文件定义了如何将
检索到的文档（RAG 结果）、对话历史、用户问题组装成一个完整的 Prompt。

核心原理：
  - Prompt 模板包含占位符（如 {documents}、{content}、{date}）
  - 图执行时，上游节点的输出会自动填充这些占位符
  - 最终生成的 Prompt 同时包含：角色定义 + RAG 文档 + 对话历史 + 用户问题

本文件的角色：
  定义 ChatTemplate 节点的实现——组装最终发送给 LLM 的 Prompt。
  这是 RAG 检索路径和对话上下文路径的汇聚点。

关键数据流：
  MilvusRetriever 输出的文档 → 填充 {documents}
  InputToChat 输出的上下文  → 填充 {content}、{history}、{date}

关联文件：
  - orchestration.go — 将本节点加入 DAG 图
  - lambda_func.go — 提供 {content}、{date}、{history} 的数据来源
*/
package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// ChatTemplateConfig 定义 Prompt 模板的配置
//
// 【AI 概念】Prompt 模板配置
//   - FormatType: 格式化类型（FString = Python 风格的 {variable} 占位符）
//   - Templates: 消息模板列表（系统消息 + 历史占位 + 用户消息）
type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

// newChatTemplate 创建 Prompt 模板节点
//
// 【AI 概念】RAG Prompt 的组装方式
// 模板由三部分组成，对应发给 LLM 的三条消息：
//
//   1. SystemMessage（系统消息）— 定义 AI 的角色、能力、输出要求
//      包含 {documents} 和 {date} 占位符，由上游节点填充
//
//   2. MessagesPlaceholder("history") — 对话历史占位符
//      会被替换为之前的 user/assistant 消息对
//
//   3. UserMessage("{content}") — 用户当前问题
//      {content} 会被替换为用户输入的问题文本
func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	config := &ChatTemplateConfig{
		FormatType: schema.FString, // 使用 {variable} 风格的占位符
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),           // 系统消息（含 {documents} 和 {date} 占位符）
			schema.MessagesPlaceholder("history", false), // 对话历史占位符（inplace=false = 保留原始格式）
			schema.UserMessage("{content}"),              // 用户消息占位符
		},
	}
	// FromMessages 将消息模板列表编译为可执行的 ChatTemplate
	// 执行时会用上游节点的输出数据替换占位符
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}

// systemPrompt 是系统提示词模板
//
// 【AI 概念】System Prompt 设计要点：
//   - 角色定义：告诉 LLM 它是谁、能做什么
//   - 上下文注入：通过占位符注入 RAG 检索结果和当前时间
//   - 输出约束：限制输出格式（本项目要求纯文本，不用 Markdown）
//
// 占位符说明：
//   - {date}      ← 来自 InputToChat 节点（lambda_func.go）
//   - {documents} ← 来自 MilvusRetriever 节点（retriever.go）
var systemPrompt = `
# 角色：对话小助手
## 核心能力
- 上下文理解与对话
- 搜索网络获得信息
## 互动指南
- 在回复前，请确保你：
  • 完全理解用户的需求和问题，如果有不清楚的地方，要向用户确认
  • 考虑最合适的解决方案方法
  • 日志主题地域：ap-guangzhou；日志主题id：869830db-a055-4479-963b-3c898d27e755
- 提供帮助时：
  • 语言清晰简洁
  • 适当的时候提供实际例子
  • 有帮助时参考文档
  • 适用时建议改进或下一步操作
- 如果请求超出了你的能力范围：
  • 清晰地说明你的局限性，如果可能的话，建议其他方法
- 如果问题是复合或复杂的，你需要一步步思考，避免直接给出质量不高的回答。
## 输出要求：
  • 易读，结构良好，必要时换行
  • 输出不能包含markdown的语法，输出需要纯文本
## 上下文信息
- 当前日期：{date}
- 相关文档：|-
==== 文档开始 ====
  {documents}
==== 文档结束 ====
  `
