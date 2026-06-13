package chat

import (
	"OpsPilot/internal/ai/agent/chat_pipeline"
	"OpsPilot/utility/log_call_back"
	"OpsPilot/utility/mem"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// ChatStream SSE 流式对话
// @Summary      流式对话
// @Description  SSE 流式 RAG 对话，逐 token 推送
// @Tags         chat
// @Accept       json
// @Produce      text/event-stream
// @Param        req  body      ChatStreamReq  true  "流式对话请求"
// @Success      200  {string}  string  "SSE stream"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /chat_stream [post]
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req ChatStreamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	id := req.Id
	msg := req.Question

	// 步骤1：创建 SSE 连接
	client, err := h.service.Create(c.Request.Context(), c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	// 步骤2：构建用户消息（包含从内存加载的对话历史）
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: mem.GetSimpleMemory(id).GetMessages(),
	}

	// 步骤3：构建 Chat Agent Pipeline 并使用同步 Invoke 执行
	// 使用 Invoke 而非 Stream，因为 Eino ReAct Agent 的 Stream 模式
	// 在 LLM 先输出文本再输出 tool_calls 时会错误终止 ReAct 循环。
	// 同步模式能正确执行完整的工具调用循环。
	runner, err := chat_pipeline.GetChatAgent(c.Request.Context())
	if err != nil {
		client.SendToClient("error", err.Error())
		return
	}

	// 发送"思考中"提示，让用户知道 Agent 正在工作
	client.SendToClient("message", "思考中")

	result, err := runner.Invoke(c.Request.Context(), userMessage, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		client.SendToClient("error", err.Error())
		return
	}

	// 步骤4：将完整回答逐片段通过 SSE 推送，模拟流式输出
	completeResponse := result.Content

	// 保存对话记忆
	if completeResponse != "" {
		mem.GetSimpleMemory(id).SetMessages(schema.UserMessage(msg))
		mem.GetSimpleMemory(id).SetMessages(schema.SystemMessage(completeResponse))
	}

	// 按标点符号分割文本，逐段推送，实现流畅的"打字机效果"
	chunks := splitForStreaming(completeResponse)
	for _, chunk := range chunks {
		client.SendToClient("message", chunk)
		time.Sleep(30 * time.Millisecond) // 控制推送节奏
	}

	client.SendToClient("done", "Stream completed")
}

// splitForStreaming 将文本按适合流式输出的粒度分割
// 优先在标点、换行处分割，保持语义完整
func splitForStreaming(text string) []string {
	if text == "" {
		return nil
	}

	var chunks []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// 在标点符号处分割
		if isSplitPoint(r) || current.Len() >= 8 {
			chunks = append(chunks, current.String())
			current.Reset()
			continue
		}

		// 最后一个字符时 flush
		if i == len([]rune(text))-1 && current.Len() > 0 {
			chunks = append(chunks, current.String())
		}
	}

	return chunks
}

// isSplitPoint 判断是否是适合分割的标点或字符
func isSplitPoint(r rune) bool {
	switch r {
	case '，', '。', '！', '？', '；', '：', '、', '\n',
		',', '.', '!', '?', ';', ':', ' ',
		'）', '】', '》', ')', ']', '>',
		'（', '【', '《', '(', '[', '<':
		return true
	}
	return false
}
