package openai

import (
	"awesome/internal/inf/ai/config"
	"awesome/internal/inf/ai/openai/audit"
	"awesome/internal/inf/ai/openai/metrics"
	"awesome/internal/inf/ai/openai/retry"
	"awesome/internal/inf/ai/openai/security"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

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
func (c *Client) StreamChatCompletion(ctx context.Context, req *ChatCompletionRequest) (<-chan ChatCompletionResponse, error) {
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

	chunkChan := make(chan ChatCompletionResponse, 100)

	go func() {
		defer close(chunkChan)
		defer httpResp.Body.Close()

		decoder := NewSSEDecoder(httpResp.Body)
		// 流式解码：每解码一个 chunk 立即发送到 channel，客户端可实时接收
		if err := decoder.DecodeOllamaResp(chunkChan); err != nil {
			// io.EOF 或网络中断，正常结束
			return
		}
	}()

	return chunkChan, nil
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

// Close 关闭客户端
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
