package chat

import (
	"OpsPilot/internal/ai/agent/chat_pipeline"
	"OpsPilot/utility/log_call_back"
	"OpsPilot/utility/mem"
	"net/http"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// Chat 同步 RAG 对话
// @Summary      对话
// @Description  同步 RAG 对话，返回完整回答
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        req  body      ChatReq  true  "对话请求"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /chat [post]
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "data": nil})
		return
	}

	id := req.Id
	msg := req.Question

	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: mem.GetSimpleMemory(id).GetMessages(),
	}

	runner, err := chat_pipeline.GetChatAgent(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error(), "data": nil})
		return
	}

	out, err := runner.Invoke(c.Request.Context(), userMessage, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error(), "data": nil})
		return
	}

	// 保存对话记忆
	mem.GetSimpleMemory(id).SetMessages(schema.UserMessage(msg))
	mem.GetSimpleMemory(id).SetMessages(schema.SystemMessage(out.Content))

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{
			"answer": out.Content,
		},
	})
}
