/*
=== AI Agent 概念：SSE（Server-Sent Events）流式输出 ===

SSE 是一种轻量级的服务端推送协议。在 AI Agent 场景中，
LLM 的响应通常是逐步生成的（token by token），SSE 可以将每个
生成片段实时推送给前端，让用户看到"打字机效果"。

核心原理：
  - HTTP 长连接：客户端发起请求后，连接保持不断开
  - 服务端持续发送事件：每个事件有 id、event、data 三个字段
  - 前端用 EventSource API 接收事件
  - 格式：text/event-stream，每条消息以 \n\n 结尾

本文件的角色：
  提供 SSE 服务端实现——管理客户端连接、发送 SSE 事件。
  这是 Agent 流式输出的基础设施层。

SSE 事件格式：
  id: 1234567890              ← 事件 ID（纳秒时间戳）
  event: message              ← 事件类型（connected / message / done / error）
  data: {"content": "你好"}   ← 事件数据（JSON 字符串）

关联文件：
  - chat_stream.go — Controller 层，调用 SSE 服务发送 Agent 响应
  - OpsPilotFrontend/app.js — 前端 EventSource 消费 SSE 事件
*/
package sse

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Client 表示一个 SSE 客户端连接
//
// 【AI 概念】SSE 客户端管理
// 每个 SSE 连接对应一个 Client 实例。
// 通过 Gin 的 ResponseWriter 向客户端推送事件。
type Client struct {
	Id     string       // 客户端唯一标识（用于关联对话记忆）
	writer http.ResponseWriter
	ctx    context.Context
}

// Service SSE 服务，管理所有客户端连接
type Service struct {
	clients *sync.Map // 线程安全的 Map，存储所有客户端连接
}

// New 创建 SSE 服务实例
func New() *Service {
	return &Service{
		clients: &sync.Map{},
	}
}

// Create 创建 SSE 连接
//
// 【AI 概念】SSE 连接建立过程
//
// 1. 设置 SSE 必需的 HTTP 响应头：
//    - Content-Type: text/event-stream  ← SSE 协议要求
//    - Cache-Control: no-cache          ← 禁止缓存实时数据
//    - Connection: keep-alive           ← 保持长连接
//    - Access-Control-Allow-Origin: *   ← CORS 跨域
//
// 2. 创建 Client 实例
//
// 3. 发送 "connected" 事件，告知前端连接已建立
func (s *Service) Create(ctx context.Context, c *gin.Context) (*Client, error) {
	// 设置 SSE 必要的 HTTP 头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建新客户端
	// client_id 优先从请求参数获取，否则自动生成 UUID
	clientId := c.Query("client_id")
	if clientId == "" {
		clientId = uuid.New().String()
	}
	client := &Client{
		Id:     clientId,
		writer: c.Writer,
		ctx:    ctx,
	}

	// 发送连接成功消息（SSE 事件格式）
	// 前端 EventSource 会触发 "connected" 事件监听器
	fmt.Fprintf(c.Writer, "id: %s\nevent: connected\ndata: {\"status\": \"connected\", \"client_id\": \"%s\"}\n\n", clientId, clientId)
	c.Writer.Flush() // 立即刷新缓冲区，确保前端收到

	return client, nil
}

// SendToClient 向指定客户端发送 SSE 事件
//
// 【AI 概念】SSE 事件格式
// 每个事件由三行组成：
//
//   id: 1700000000000000000  ← 唯一事件 ID（纳秒时间戳）
//   event: message           ← 事件类型
//   data: 你好，我是助手       ← 事件数据
//   （空行）                  ← 事件之间用空行分隔
//
// 事件类型约定：
//   - "connected" : 连接已建立
//   - "message"   : Agent 响应片段（逐 token 推送）
//   - "done"      : 响应完成
//   - "error"     : 发生错误
func (c *Client) SendToClient(eventType, data string) bool {
	msg := fmt.Sprintf(
		"id: %d\nevent: %s\ndata: %s\n\n",
		time.Now().UnixNano(), // 用纳秒时间戳作为事件 ID
		eventType,
		data,
	)
	// 通过 ResponseWriter 直接写入 SSE 事件
	c.writer.Write([]byte(msg))
	if f, ok := c.writer.(http.Flusher); ok {
		f.Flush() // 刷新缓冲区，立即发送给前端
	}
	return true
}
