/*
=== AI Agent 概念：图节点数据转换（Lambda 函数）===

在 DAG 图中，不同节点需要不同格式的数据。Lambda 函数就是节点间的
"数据适配器"，负责将上游输出转换为下游期望的格式。

核心原理：
  - 图的输入类型是 UserMessage（统一结构）
  - 但每个下游节点需要不同格式的数据：
    - MilvusRetriever 需要纯文本（string）
    - ChatTemplate 需要 map（包含多个字段的字典）
  - Lambda 函数完成这些格式转换

本文件的角色：
  定义两个 Lambda 转换函数，分别服务于 RAG 检索路径和对话上下文路径。

关键数据流：
  UserMessage ──→ InputToRag  ──→ string（问题文本）──→ MilvusRetriever
  UserMessage ──→ InputToChat ──→ map{content, history, date} ──→ ChatTemplate

关联文件：
  - orchestration.go — 将这两个 Lambda 注册为图节点
  - types.go — UserMessage 结构体定义
  - prompt.go — 使用 {content}、{history}、{date} 占位符
*/
package chat_pipeline

import (
	"context"
	"time"
)

// newInputToRagLambda 从 UserMessage 中提取纯文本问题，用于 RAG 检索
//
// 【AI 概念】数据转换 Lambda
// MilvusRetriever 节点需要纯文本（string）作为输入进行向量检索。
// 此函数从 UserMessage 中提取 Query 字段，丢弃 ID 和 History。
//
// 输入：*UserMessage{ID, Query, History}
// 输出：string（Query 字段的值）
func newInputToRagLambda(ctx context.Context, input *UserMessage, opts ...any) (output string, err error) {
	return input.Query, nil
}

// newInputToChatLambda 从 UserMessage 中提取对话上下文，用于 Prompt 组装
//
// 【AI 概念】多字段数据转换
// ChatTemplate 需要多个字段来填充 Prompt 模板的占位符：
//   - "content" → 填充 UserMessage("{content}") 中的 {content}
//   - "history" → 填充 MessagesPlaceholder("history") 中的对话历史
//   - "date"    → 填充 systemPrompt 中的 {date}
//
// 输入：*UserMessage{ID, Query, History}
// 输出：map[string]any{"content": 问题, "history": 历史消息, "date": 格式化时间}
func newInputToChatLambda(ctx context.Context, input *UserMessage, opts ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,                        // 用户当前问题
		"history": input.History,                      // 之前的对话历史（[]*schema.Message）
		"date":    time.Now().Format("2006-01-02 15:04:05"), // 当前时间（Go 的时间格式化参考时间）
	}, nil
}
