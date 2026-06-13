package chat

import (
	"OpsPilot/internal/logic/sse"
)

// ChatHandler 处理所有聊天相关的 HTTP 请求
type ChatHandler struct {
	service *sse.Service
}

// NewChatHandler 创建 ChatHandler 实例
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		service: sse.New(),
	}
}
