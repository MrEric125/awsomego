package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"awesome/internal/ai/openai"
	"awesome/internal/ai/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// EnterpriseAIHandler 企业级 AI 处理器
type EnterpriseAIHandler struct {
	enterpriseService *service.EnterpriseAIService
	logger            *zap.SugaredLogger
}

// NewEnterpriseAIHandler 创建企业级 AI 处理器
func NewEnterpriseAIHandler(svc *service.EnterpriseAIService) *EnterpriseAIHandler {
	logger, _ := zap.NewProduction()
	return &EnterpriseAIHandler{
		enterpriseService: svc,
		logger:            logger.Sugar(),
	}
}

// RegisterRoutes 注册路由
func (h *EnterpriseAIHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	{
		// 聊天接口
		v1.POST("/chat/completions", h.ChatCompletions)
		v1.POST("/chat/completions/stream", h.ChatCompletionsStream)

		// 嵌入接口
		v1.POST("/embeddings", h.Embeddings)

		// 健康检查
		v1.GET("/health", h.Health)

		// 统计信息
		v1.GET("/stats", h.Stats)
	}

	// API 文档
	docs := r.Group("/docs")
	{
		docs.GET("/openapi.json", h.OpenAPISpec)
		docs.GET("/api", h.APIDocs)
	}
}

// ChatCompletionsRequest 聊天完成请求
type ChatCompletionsRequest struct {
	Model            string                 `json:"model" binding:"required"`
	Messages         []openai.ChatMessage   `json:"messages" binding:"required"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	N                *int                   `json:"n,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
	ResponseFormat   string                 `json:"response_format,omitempty"`
	UseCache         bool                   `json:"use_cache,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ChatCompletionsResponse 聊天完成响应
type ChatCompletionsResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	Created       int64                  `json:"created"`
	Model         string                 `json:"model"`
	Choices       []ChoiceResponse       `json:"choices"`
	Usage         *UsageResponse         `json:"usage,omitempty"`
	SystemFingerprint string              `json:"system_fingerprint,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Cached        bool                   `json:"cached"`
	LatencyMs     int64                  `json:"latency_ms"`
}

// ChoiceResponse 选择响应
type ChoiceResponse struct {
	Index        int            `json:"index"`
	Message      MessageResponse `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// UsageResponse 使用量响应
type UsageResponse struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Param      string `json:"param,omitempty"`
	HTTPStatus int    `json:"-"`
}

// ChatCompletions 处理聊天完成请求
func (h *EnterpriseAIHandler) ChatCompletions(c *gin.Context) {
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	// 验证请求
	if err := h.validateChatRequest(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// 构建服务请求
	svcReq := &service.ChatRequest{
		Messages:     req.Messages,
		Model:        req.Model,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		TopP:         req.TopP,
		ResponseFormat: req.ResponseFormat,
		User:         req.User,
		UseCache:     req.UseCache,
	}

	// 添加请求上下文
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, "request_id", generateRequestID())
	ctx = context.WithValue(ctx, "user_id", req.User)
	ctx = context.WithValue(ctx, "ip_address", c.ClientIP())

	// 调用服务
	resp, err := h.enterpriseService.Chat(ctx, svcReq)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// 构建响应
	result := ChatCompletionsResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: []ChoiceResponse{
			{
				Index: 0,
				Message: MessageResponse{
					Role:    "assistant",
					Content: resp.Content,
				},
				FinishReason: resp.FinishReason,
			},
		},
		Cached:    resp.Cached,
		LatencyMs: resp.LatencyMs,
	}

	if resp.Usage != nil {
		result.Usage = &UsageResponse{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	c.JSON(http.StatusOK, result)
}

// ChatCompletionsStream 处理流式聊天完成请求
func (h *EnterpriseAIHandler) ChatCompletionsStream(c *gin.Context) {
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	// 验证请求
	if err := h.validateChatRequest(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// 构建服务请求
	svcReq := &service.ChatRequest{
		Messages:     req.Messages,
		Model:        req.Model,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		TopP:         req.TopP,
		User:         req.User,
	}

	// 添加请求上下文
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, "request_id", generateRequestID())
	ctx = context.WithValue(ctx, "user_id", req.User)
	ctx = context.WithValue(ctx, "ip_address", c.ClientIP())

	// 调用流式服务
	stream, err := h.enterpriseService.StreamChat(ctx, svcReq)
	if err != nil {
		h.handleError(c, err)
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

		if chunk.Error != nil {
			h.logger.Errorw("stream error", "error", chunk.Error)
			return false
		}

		// 构建 SSE 数据
		data := map[string]interface{}{
			"id":      generateRequestID(),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{
						"content": chunk.Content,
					},
					"finish_reason": chunk.FinishReason,
				},
			},
		}

		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
		c.Writer.Flush()

		return true
	})
}

// EmbeddingsRequest 嵌入请求
type EmbeddingsRequest struct {
	Model string   `json:"model,omitempty"`
	Input []string `json:"input" binding:"required"`
	User  string   `json:"user,omitempty"`
}

// EmbeddingsResponse 嵌入响应
type EmbeddingsResponse struct {
	Object  string               `json:"object"`
	Data    []EmbeddingDataResponse `json:"data"`
	Model   string               `json:"model"`
	Usage   *UsageResponse       `json:"usage"`
	LatencyMs int64              `json:"latency_ms"`
}

// EmbeddingDataResponse 嵌入数据响应
type EmbeddingDataResponse struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// Embeddings 处理嵌入请求
func (h *EnterpriseAIHandler) Embeddings(c *gin.Context) {
	var req EmbeddingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	// 构建服务请求
	svcReq := &service.EmbeddingRequest{
		Input: req.Input,
		Model: req.Model,
		User:  req.User,
	}

	// 调用服务
	ctx := c.Request.Context()
	resp, err := h.enterpriseService.CreateEmbedding(ctx, svcReq)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// 构建响应
	data := make([]EmbeddingDataResponse, len(resp.Data))
	for i, d := range resp.Data {
		data[i] = EmbeddingDataResponse{
			Object:    d.Object,
			Index:     d.Index,
			Embedding: d.Embedding,
		}
	}

	result := EmbeddingsResponse{
		Object:    resp.Object,
		Data:      data,
		Model:     resp.Model,
		LatencyMs: resp.LatencyMs,
	}

	if resp.Usage != nil {
		result.Usage = &UsageResponse{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	c.JSON(http.StatusOK, result)
}

// Health 健康检查
func (h *EnterpriseAIHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "enterprise-ai",
	})
}

// Stats 统计信息
func (h *EnterpriseAIHandler) Stats(c *gin.Context) {
	stats := h.enterpriseService.GetStats()
	c.JSON(http.StatusOK, stats)
}

// OpenAPISpec OpenAPI 规范
func (h *EnterpriseAIHandler) OpenAPISpec(c *gin.Context) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Enterprise AI API",
			"description": "企业级 AI 服务 API，支持 OpenAI 兼容接口",
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{"url": "/v1", "description": "API v1"},
		},
		"paths": map[string]interface{}{
			"/chat/completions": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "聊天完成",
					"description": "创建聊天完成请求，支持多种模型",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/ChatCompletionsRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "成功响应",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{
										"$ref": "#/components/schemas/ChatCompletionsResponse",
									},
								},
							},
						},
					},
				},
			},
			"/embeddings": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "创建嵌入",
					"description": "为文本创建向量嵌入",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/EmbeddingsRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "成功响应",
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"ChatCompletionsRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"model", "messages"},
					"properties": map[string]interface{}{
						"model": map[string]string{
							"type":        "string",
							"description": "模型名称，如 gpt-4, gpt-3.5-turbo",
							"example":     "gpt-4",
						},
						"messages": map[string]interface{}{
							"type":        "array",
							"description": "对话消息列表",
							"items": map[string]string{
								"$ref": "#/components/schemas/ChatMessage",
							},
						},
						"temperature": map[string]string{
							"type":        "number",
							"description": "采样温度，0-2",
							"default":     "0.7",
						},
						"max_tokens": map[string]string{
							"type":        "integer",
							"description": "最大生成令牌数",
						},
						"use_cache": map[string]string{
							"type":        "boolean",
							"description": "是否使用缓存",
							"default":     "false",
						},
					},
				},
				"ChatMessage": map[string]interface{}{
					"type": "object",
					"required": []string{"role", "content"},
					"properties": map[string]interface{}{
						"role": map[string]interface{}{
							"type":        "string",
							"enum":        []interface{}{"system", "user", "assistant"},
							"description": "消息角色",
						},
						"content": map[string]string{
							"type":        "string",
							"description": "消息内容",
						},
					},
				},
			},
		},
	}

	c.JSON(http.StatusOK, spec)
}

// APIDocs API 文档页面
func (h *EnterpriseAIHandler) APIDocs(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Enterprise AI API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4/swagger-ui-bundle.js"></script>
    <script>
        const ui = SwaggerUIBundle({
            url: '/docs/openapi.json',
            dom_id: '#swagger-ui',
        })
    </script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// validateChatRequest 验证聊天请求
func (h *EnterpriseAIHandler) validateChatRequest(req *ChatCompletionsRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages is required")
	}
	for i, msg := range req.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message %d: role is required", i)
		}
		if msg.Content == "" {
			return fmt.Errorf("message %d: content is required", i)
		}
		if msg.Role != "system" && msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("message %d: invalid role %s", i, msg.Role)
		}
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if req.MaxTokens != nil && *req.MaxTokens < 1 {
		return fmt.Errorf("max_tokens must be positive")
	}
	return nil
}

// handleError 处理错误
func (h *EnterpriseAIHandler) handleError(c *gin.Context, err error) {
	if apiErr, ok := err.(*openai.APIError); ok {
		status := apiErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}
		h.sendError(c, status, apiErr.Code, apiErr.Message)
		return
	}

	// 检查是否为限流错误
	if strings.Contains(err.Error(), "rate limit") {
		h.sendError(c, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error())
		return
	}

	// 检查是否为超时
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
		h.sendError(c, http.StatusGatewayTimeout, "timeout", err.Error())
		return
	}

	h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
}

// sendError 发送错误响应
func (h *EnterpriseAIHandler) sendError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:       code,
			Message:    message,
			Type:       "api_error",
			HTTPStatus: status,
		},
	})
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
