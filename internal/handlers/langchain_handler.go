package handlers

import (
	"awesome/internal/inf/ai/new/langchain/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LangChainHandler LangChain HTTP 处理器
type LangChainHandler struct {
	lcService service.LangChainService
}

// NewLangChainHandler 创建 LangChain 处理器
func NewLangChainHandler(lcService service.LangChainService) *LangChainHandler {
	return &LangChainHandler{
		lcService: lcService,
	}
}

// ChatRequest 对话请求
type ChatRequest struct {
	Message string            `json:"message" binding:"required"`
	History []service.Message `json:"history,omitempty"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Reply   string `json:"reply"`
	Created int64  `json:"created"`
}

// SummarizeRequest 摘要请求
type SummarizeRequest struct {
	Text string `json:"text" binding:"required"`
}

// SummarizeResponse 摘要响应
type SummarizeResponse struct {
	Summary string `json:"summary"`
}

// TranslateRequest 翻译请求
type TranslateRequest struct {
	Text       string `json:"text" binding:"required"`
	TargetLang string `json:"target_lang" binding:"required"`
}

// TranslateResponse 翻译响应
type TranslateResponse struct {
	Translation string `json:"translation"`
}

// ChainRequest 链请求
type ChainRequest struct {
	Prompt    string            `json:"prompt" binding:"required"`
	ChainType service.ChainType `json:"chain_type" binding:"required"`
}

// ChainResponse 链响应
type ChainResponse struct {
	Result string `json:"result"`
}

// QARequest 问答请求
type QARequest struct {
	Question  string   `json:"question" binding:"required"`
	Documents []string `json:"documents" binding:"required"`
}

// QAResponse 问答响应
type QAResponse struct {
	Answer string `json:"answer"`
}

// Chat 处理对话请求
func (h *LangChainHandler) Chat(c *gin.Context) {
	var req ChatRequest
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
		reply, err = h.lcService.ChatWithHistory(ctx, req.History)
	} else {
		reply, err = h.lcService.Chat(ctx, req.Message)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Chat failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		Reply:   reply,
		Created: time.Now().Unix(),
	})
}

// RunChain 处理链执行请求
func (h *LangChainHandler) RunChain(c *gin.Context) {
	var req ChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	result, err := h.lcService.RunChain(ctx, req.Prompt, req.ChainType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Chain execution failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ChainResponse{
		Result: result,
	})
}

// Summarize 处理摘要请求
func (h *LangChainHandler) Summarize(c *gin.Context) {
	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	summary, err := h.lcService.Summarize(ctx, req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Summarization failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SummarizeResponse{
		Summary: summary,
	})
}

// Translate 处理翻译请求
func (h *LangChainHandler) Translate(c *gin.Context) {
	var req TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	translation, err := h.lcService.Translate(ctx, req.Text, req.TargetLang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Translation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TranslateResponse{
		Translation: translation,
	})
}

// QuestionAnswering 处理问答请求
func (h *LangChainHandler) QuestionAnswering(c *gin.Context) {
	var req QARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	answer, err := h.lcService.QuestionAnswering(ctx, req.Question, req.Documents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Question answering failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, QAResponse{
		Answer: answer,
	})
}
