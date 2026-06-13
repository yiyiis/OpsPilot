/*
=== AI Agent 概念：Pipeline 输入数据类型 ===

本文件定义了 RAG Chat Pipeline 的核心输入数据结构。
整个 DAG 图的入口类型就是 *UserMessage，所有节点从这个结构中提取所需的数据。

关联文件：
  - orchestration.go — 使用 UserMessage 作为图的输入类型
  - lambda_func.go — 从 UserMessage 中提取各节点所需的数据
*/
package chat_pipeline

import "github.com/cloudwego/eino/schema"

// UserMessage 是 RAG Chat Pipeline 的输入类型
//
// 【数据流】前端请求 → Controller 构建 UserMessage → 传入 DAG 图
//
// 字段说明：
//   - ID: 会话标识，用于关联对话记忆（mem 包使用）
//   - Query: 用户当前提出的问题
//   - History: 之前的对话历史（user/assistant 消息对），用于多轮对话上下文
type UserMessage struct {
	ID      string            `json:"id"`
	Query   string            `json:"query"`
	History []*schema.Message `json:"history"`
}
