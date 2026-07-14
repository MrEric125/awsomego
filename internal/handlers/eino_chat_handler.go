package handlers

import (
	"awesome/internal/inf/ai/new/model"
	"awesome/internal/inf/ai/new/service"
	"awesome/internal/inf/logger"
	"awesome/internal/response"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChatHandler struct {
	service *service.ChatService
	//logger  *zap.Logger
}

func NewChatHandler(service *service.ChatService) *ChatHandler {
	return &ChatHandler{
		service: service,
	}
}

// RegisterRoutes 注册 AI 路由
func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup) {
	ai := r.Group("/ai")
	{
		ai.POST("/chat", h.ChatCompletion)
		ai.POST("/chat/stream", h.ChatCompletionStream)
		ai.POST("/listModels", h.ListModels)
		//ai.POST("/summarize", h.Summarize)
		//ai.POST("/translate", h.Translate)
	}
}

// ChatCompletion 普通对话接口
func (h *ChatHandler) ChatCompletion(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Error("invalid request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// 检查是否为流式请求
	if req.Stream {
		h.handleStream(c, &req)
		return
	}

	// 调用服务层
	chatResponse, err := h.service.Chat(c.Request.Context(), &req)
	if err != nil {
		logger.Logger.Error("chat completion failed", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "chat failed: "+err.Error())
		return
	}

	response.Success(c, chatResponse)
}

// ChatCompletionStream 流式对话接口
func (h *ChatHandler) ChatCompletionStream(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Error("invalid stream request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	h.handleStream(c, &req)
}

// handleStream 处理流式响应（内部方法，避免重复读取请求体）
func (h *ChatHandler) handleStream(c *gin.Context, req *model.ChatRequest) {
	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 调用流式服务
	streamChan, err := h.service.ChatStream(c.Request.Context(), req)
	if err != nil {
		logger.Logger.Error("stream chat failed", zap.Error(err))
		h.sendSSEError(c, err.Error())
		return
	}

	c.Stream(func(w io.Writer) bool {
		msg, err := streamChan.Recv()
		if err == io.EOF {
			h.sendSSEDone(c)
			return false
		}
		if err != nil {
			logger.Logger.Error("stream read error", zap.Error(err))
			h.sendSSEError(c, err.Error())
			return false
		}

		// 转换流式 tool_calls
		var deltaToolCalls []model.ToolCallInfo
		if len(msg.ToolCalls) > 0 {
			deltaToolCalls = make([]model.ToolCallInfo, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				deltaToolCalls[i] = model.ToolCallInfo{
					ID:   tc.ID,
					Type: tc.Type,
					Function: model.FunctionCallInfo{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		// 构造SSE事件
		event := model.StreamEvent{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []model.Choice{
				{
					Index: 0,
					Delta: model.Message{
						Role:      string(msg.Role),
						Content:   msg.Content,
						ToolCalls: deltaToolCalls,
					},
				},
			},
		}

		data, _ := json.Marshal(event)
		c.SSEvent("message", string(data))
		return true
	})
}

func (h *ChatHandler) sendSSEDone(c *gin.Context) {
	c.SSEvent("message", "[DONE]")
}

func (h *ChatHandler) sendSSEError(c *gin.Context, errMsg string) {
	event := model.StreamEvent{
		Error: errMsg,
	}
	data, _ := json.Marshal(event)
	c.SSEvent("error", string(data))
}

// ListModels 列出支持的模型
func (h *ChatHandler) ListModels(c *gin.Context) {
	models := h.service.ListModels()
	response.Success(c, gin.H{
		"models": models,
	})
}
