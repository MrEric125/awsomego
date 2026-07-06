package openai

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"awesome/internal/ai/config"
	"awesome/internal/ai/openai/audit"
	"awesome/internal/ai/openai/metrics"
	"awesome/internal/ai/openai/retry"
	"awesome/internal/ai/openai/security"

	"go.uber.org/zap"
)

// Client OpenAI 客户端
type Client struct {
	config      *config.AIConfig
	httpClient  *http.Client
	retryPolicy *retry.Policy
	security    *security.Filter
	audit       *audit.Logger
	metrics     *metrics.Collector
	logger      *zap.SugaredLogger

	// 连接池管理
	connectionPool *ConnectionPool

	// 请求计数器
	requestCount int64
	successCount int64
	failureCount int64

	// 熔断器
	circuitBreaker *CircuitBreaker

	mu sync.RWMutex
}

// ClientOption 客户端选项
type ClientOption func(*Client)

// NewClient 创建 OpenAI 客户端
func NewClient(cfg *config.AIConfig, opts ...ClientOption) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.OpenAIBaseURL == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// 创建 HTTP 客户端
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableCompression: false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	// 创建日志器
	logger, _ := zap.NewProduction()
	sugarLogger := logger.Sugar()

	client := &Client{
		config:     cfg,
		httpClient: httpClient,
		retryPolicy: retry.NewPolicy(
			retry.WithMaxRetries(cfg.MaxRetries),
			retry.WithInitialDelay(100*time.Millisecond),
			retry.WithMaxDelay(5*time.Second),
			retry.WithMultiplier(2.0),
		),
		security:       security.NewFilter(),
		audit:          audit.NewLogger(sugarLogger),
		metrics:        metrics.NewCollector(),
		logger:         sugarLogger,
		connectionPool: NewConnectionPool(20),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	N                *int                   `json:"n,omitempty"`
	Stream           bool                   `json:"stream"`
	Stop             interface{}            `json:"stop,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64     `json:"logit_bias,omitempty"`
	User             string                 `json:"user,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// Tool 工具定义
type Tool struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ResponseFormat 响应格式
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatCompletionResponse 聊天完成响应
//type ChatCompletionResponse struct {
//	ID                string                 `json:"id"`
//	Object            string                 `json:"object"`
//	Created           int64                  `json:"created"`
//	Model             string                 `json:"model"`
//	Choices           []ChatChoice           `json:"choices"`
//	Usage             Usage                  `json:"usage"`
//	SystemFingerprint string                 `json:"system_fingerprint"`
//	Error             *APIError              `json:"error,omitempty"`
//	Metadata          map[string]interface{} `json:"metadata,omitempty"`
//}

type ChatCompletionResponse struct {
	Model              string      `json:"model"`
	CreatedAt          time.Time   `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	DoneReason         string      `json:"done_reason"`
	TotalDuration      int64       `json:"total_duration"`
	LoadDuration       int64       `json:"load_duration"`
	PromptEvalCount    int         `json:"prompt_eval_count"`
	PromptEvalDuration int         `json:"prompt_eval_duration"`
	EvalCount          int         `json:"eval_count"`
	EvalDuration       int         `json:"eval_duration"`
}

// ChatChoice 聊天选择
type ChatChoice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason"`
	Logprobs     interface{}  `json:"logprobs,omitempty"`
}

// Usage 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError API 错误
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Param      string `json:"param"`
	HTTPStatus int    `json:"-"`
}

// Error 实现 error 接口
func (e *APIError) Error() string {
	return fmt.Sprintf("OpenAI API error: %s (code: %s, type: %s)", e.Message, e.Code, e.Type)
}

// ChatCompletion 聊天完成
func (c *Client) ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// 增加请求计数
	atomic.AddInt64(&c.requestCount, 1)

	// 检查熔断器
	if !c.circuitBreaker.Allow() {
		atomic.AddInt64(&c.failureCount, 1)
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// 安全过滤
	for i := range req.Messages {
		filtered, warnings := c.security.Filter(req.Messages[i].Content)
		if len(warnings) > 0 {
			c.logger.Warnw("sensitive content detected",
				"warnings", warnings,
				"role", req.Messages[i].Role,
			)
		}
		req.Messages[i].Content = filtered
	}

	// 设置默认模型
	if req.Model == "" {
		req.Model = c.config.OpenAIModelName
	}

	// 设置默认参数
	if req.Temperature == nil {
		temp := c.config.Temperature
		req.Temperature = &temp
	}
	if req.MaxTokens == nil {
		maxTokens := c.config.MaxTokens
		req.MaxTokens = &maxTokens
	}

	// 记录审计日志
	c.audit.LogRequest(ctx, req)

	// 执行请求（带重试）
	var resp *ChatCompletionResponse
	var err error

	startTime := time.Now()
	err = c.retryPolicy.Execute(ctx, func() error {
		resp, err = c.doChatCompletion(ctx, req)
		return err
	})
	duration := time.Since(startTime)

	// 记录指标
	c.metrics.RecordRequest(duration, err == nil)

	if err != nil {
		atomic.AddInt64(&c.failureCount, 1)
		c.circuitBreaker.RecordFailure()
		c.audit.LogError(ctx, err)
		return nil, err
	}

	atomic.AddInt64(&c.successCount, 1)
	c.circuitBreaker.RecordSuccess()

	// 记录审计日志
	c.audit.LogResponse(ctx, resp)

	return resp, nil
}

// doChatCompletion 执行聊天完成请求
func (c *Client) doChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat", strings.TrimSuffix(c.config.OpenAIBaseURL, "/"))
	c.logger.Infof("chat url:%s", url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.OpenAIAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.OpenAIAPIKey))
	}

	// 从连接池获取连接
	conn := c.connectionPool.Get()
	defer c.connectionPool.Put(conn)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	c.logger.Infof("chat resp:%s", respBody)

	if httpResp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
		}
		apiErr.HTTPStatus = httpResp.StatusCode
		return nil, &apiErr
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// StreamChatCompletion 流式聊天完成
func (c *Client) StreamChatCompletion(ctx context.Context, req *ChatCompletionRequest) (<-chan StreamChunk, error) {
	// 设置流式请求
	req.Stream = true

	// 安全过滤
	for i := range req.Messages {
		filtered, warnings := c.security.Filter(req.Messages[i].Content)
		if len(warnings) > 0 {
			c.logger.Warnw("sensitive content detected",
				"warnings", warnings,
				"role", req.Messages[i].Role,
			)
		}
		req.Messages[i].Content = filtered
	}

	// 设置默认模型
	if req.Model == "" {
		req.Model = c.config.OpenAIModelName
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat", strings.TrimSuffix(c.config.OpenAIBaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.OpenAIAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.OpenAIAPIKey))

	}
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	chunkChan := make(chan StreamChunk, 100)

	go func() {
		defer close(chunkChan)
		defer httpResp.Body.Close()

		decoder := NewSSEDecoder(httpResp.Body)
		for {
			chunk, err := decoder.DecodeOllamaResp()
			if err != nil {
				if err == io.EOF {
					return
				}
				chunkChan <- StreamChunk{Error: err}
				return
			}

			if chunk != nil {
				chunkChan <- *chunk
			}
		}
	}()

	return chunkChan, nil
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID      string
	Object  string
	Created int64
	Model   string
	Choices []TrunkItem
	Error   error
}

// StreamChoice 流式选择
type StreamChoice struct {
	Index        int
	Delta        ChatMessage
	FinishReason string
}

// ollama 响应结果
type TrunkItem struct {
	Model              string      `json:"model"`
	CreatedAt          time.Time   `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	DoneReason         string      `json:"done_reason"`
	TotalDuration      int64       `json:"total_duration"`
	LoadDuration       int64       `json:"load_duration"`
	PromptEvalCount    int         `json:"prompt_eval_count"`
	PromptEvalDuration int         `json:"prompt_eval_duration"`
	EvalCount          int         `json:"eval_count"`
	EvalDuration       int         `json:"eval_duration"`
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	User  string   `json:"user,omitempty"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData 嵌入数据
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// CreateEmbedding 创建嵌入
func (c *Client) CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if req.Model == "" {
		req.Model = "text-embedding-ada-002"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", strings.TrimSuffix(c.config.OpenAIBaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.OpenAIAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.OpenAIAPIKey))
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var resp EmbeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// GetMetrics 获取指标
func (c *Client) GetMetrics() *metrics.Stats {
	return c.metrics.GetStats()
}

// GetStats 获取统计信息
func (c *Client) GetStats() ClientStats {
	return ClientStats{
		TotalRequests:  atomic.LoadInt64(&c.requestCount),
		SuccessCount:   atomic.LoadInt64(&c.successCount),
		FailureCount:   atomic.LoadInt64(&c.failureCount),
		CircuitState:   c.circuitBreaker.State(),
		ConnectionPool: c.connectionPool.Stats(),
	}
}

// ClientStats 客户端统计
type ClientStats struct {
	TotalRequests  int64
	SuccessCount   int64
	FailureCount   int64
	CircuitState   string
	ConnectionPool PoolStats
}

// Close 关闭客户端
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
