/*
=== AI Agent 概念：对话记忆管理 ===

对话记忆让 Agent 具备多轮对话能力——记住之前的对话内容，
在后续回答中参考历史上下文。

核心原理：
  - 每次对话后，将用户消息和 Agent 回答存入记忆
  - 下次对话时，从记忆中加载历史消息，作为 Prompt 的一部分
  - 使用滑动窗口策略：当消息数量超过上限时，淘汰最早的消息

本文件的角色：
  提供基于内存的对话记忆管理。使用 Go map 按会话 ID 存储记忆，
  每个会话维护一个消息列表（滑动窗口，最多 6 条）。

⚠️ 注意：当前实现使用内存存储，服务重启后记忆会丢失。
  生产环境应考虑使用 Redis 或数据库持久化。

滑动窗口策略：
  - MaxWindowSize: 6（最多保留 6 条消息）
  - 淘汰策略：成对淘汰（保持 user/assistant 消息配对）
  - 淘汰位置：从最早的消息开始丢弃

关联文件：
  - chat_v1_chat_stream.go — 保存和加载对话记忆
  - chat_pipeline/lambda_func.go — 将历史消息注入 Prompt
*/
package mem

import (
	"sync"

	"github.com/cloudwego/eino/schema"
)

// SimpleMemoryMap 全局会话记忆存储
// key: 会话 ID（前端生成的 client_id）
// value: *SimpleMemory（该会话的消息历史）
var SimpleMemoryMap = make(map[string]*SimpleMemory)

// mu 保护 SimpleMemoryMap 的并发访问
var mu sync.Mutex

// GetSimpleMemory 获取指定会话的记忆实例
//
// 【AI 概念】会话管理
// 每个前端连接有唯一的 client_id，对应一个独立的对话记忆。
// 首次访问时自动创建新的记忆实例。
//
// 返回值：*SimpleMemory（始终非 nil）
func GetSimpleMemory(id string) *SimpleMemory {
	mu.Lock()
	defer mu.Unlock()
	// 如果存在就返回，不存在就创建
	if mem, ok := SimpleMemoryMap[id]; ok {
		return mem
	} else {
		newMem := &SimpleMemory{
			ID:            id,
			Messages:      []*schema.Message{},
			MaxWindowSize: 6, // 最多保留 6 条消息（约 3 轮对话）
		}
		SimpleMemoryMap[id] = newMem
		return newMem
	}
}

// SimpleMemory 单个会话的记忆存储
//
// 【AI 概念】滑动窗口记忆
// 当消息数量超过 MaxWindowSize 时，淘汰最早的消息。
// 成对淘汰确保 user/assistant 消息始终配对，避免上下文断裂。
type SimpleMemory struct {
	ID            string            `json:"id"`      // 会话 ID
	Messages      []*schema.Message `json:"messages"` // 消息历史
	MaxWindowSize int               // 窗口大小上限
	mu            sync.Mutex        // 保护本实例的并发访问
}

// SetMessages 向记忆中追加一条消息
//
// 【AI 概念】成对淘汰策略
// 当消息数量超过 MaxWindowSize 时：
//   1. 计算超出数量（excess）
//   2. 如果 excess 是奇数，加 1 变为偶数（确保成对淘汰）
//   3. 从头部删除 excess 条消息
//
// 为什么要成对？
//   如果只丢弃 user 消息而不丢弃对应的 assistant 回答，
//   Agent 会看到"没有对应问题的回答"，导致上下文混乱。
//
// 示例（MaxWindowSize=6）：
//   当前 8 条消息 → excess=2 → 淘汰前 2 条（1 对 user/assistant）
//   当前 9 条消息 → excess=3 → +1=4 → 淘汰前 4 条（2 对）
func (c *SimpleMemory) SetMessages(msg *schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, msg)
	if len(c.Messages) > c.MaxWindowSize {
		// 计算需要丢弃的消息数量（必须是偶数，保持对话配对）
		excess := len(c.Messages) - c.MaxWindowSize
		if excess%2 != 0 {
			excess++ // 奇数变偶数
		}
		// 从头部丢弃最早的消息
		c.Messages = c.Messages[excess:]
	}
}

// GetMessages 获取当前记忆中的所有消息
//
// 返回值是内部 slice 的引用，调用者不应修改返回的 slice。
func (c *SimpleMemory) GetMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Messages
}
