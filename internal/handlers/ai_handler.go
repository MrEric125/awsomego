package handlers

import (
	handlers2 "awesome/internal/inf/ai/dto"
	"awesome/internal/inf/ai/service"
	"awesome/internal/inf/evaluation/logger"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AIHandler AI 相关 HTTP 处理器
type AIHandler struct {
	aiService service.AIService
}

// NewAIHandler 创建 AI 处理器
func NewAIHandler(aiService service.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// RegisterRoutes 注册 AI 路由
func (h *AIHandler) RegisterRoutes(r *gin.RouterGroup) {
	ai := r.Group("/ai")
	{
		ai.POST("/chat", h.Chat)
		ai.POST("/chat/stream", h.StreamChat)
		ai.POST("/summarize", h.Summarize)
		ai.POST("/translate", h.Translate)
	}
}

// Chat 处理对话请求
func (h *AIHandler) Chat(c *gin.Context) {
	var req handlers2.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	var reply string
	var err error

	// 如果有历史记录，使用带历史的对话
	if len(req.History) > 0 {
		var messages []service.Message
		for _, msg := range req.History {
			messages = append(messages, service.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
		// 添加当前消息
		messages = append(messages, service.Message{
			Role:    "user",
			Content: req.Message,
		})
		reply, err = h.aiService.ChatWithHistory(ctx, messages, req.Provider, req.Model)
	} else {
		reply, err = h.aiService.Chat(ctx, req.Message, req.Provider, req.Model)
	}

	if err != nil {
		logger.Infof("chat error:%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Chat failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, handlers2.ChatResponse{
		Reply:   reply,
		Created: time.Now().Unix(),
	})
}

// StreamChat 处理流式对话请求
func (h *AIHandler) StreamChat(c *gin.Context) {
	var req handlers2.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	stream, err := h.aiService.StreamChat(ctx, req.Message, req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Stream chat failed: " + err.Error(),
		})
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Stream(func(w io.Writer) bool {
		chunk, ok := <-stream
		if !ok {
			return false
		}

		// SSE 格式
		c.SSEvent("message", chunk)
		c.Writer.Flush()
		return true
	})
}

// Summarize 处理文本摘要请求
func (h *AIHandler) Summarize(c *gin.Context) {
	var req handlers2.SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	summary, err := h.aiService.Summarize(ctx, req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Summarization failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, handlers2.SummarizeResponse{
		Summary: summary,
	})
}

// Translate 处理翻译请求
func (h *AIHandler) Translate(c *gin.Context) {
	var req handlers2.TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	translation, err := h.aiService.Translate(ctx, req.Text, req.TargetLang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Translation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, handlers2.TranslateResponse{
		Translation: translation,
	})
}
