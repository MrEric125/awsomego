package service

import (
	"awesome/internal/inf/ai/config"
	openai2 "awesome/internal/inf/ai/openai"
	"awesome/internal/inf/ai/openai/ratelimit"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// EnterpriseAIService 企业级 AI 服务
type EnterpriseAIService struct {
	config       *config.AIConfig
	openaiClient *openai2.Client
	limiter      ratelimit.Limiter
	logger       *zap.SugaredLogger

	// 并发控制
	semaphore      chan struct{}
	maxConcurrency int

	// 统计
	requestCount int64
	successCount int64
	failureCount int64
	totalLatency int64

	// 缓存
	cache        *ResponseCache
	cacheEnabled bool

	mu sync.RWMutex
}

// EnterpriseAIServiceOption 服务选项
type EnterpriseAIServiceOption func(*EnterpriseAIService)

// NewEnterpriseAIService 创建企业级 AI 服务
func NewEnterpriseAIService(cfg *config.AIConfig, opts ...EnterpriseAIServiceOption) (*EnterpriseAIService, error) {
	// 创建 OpenAI 客户端
	client, err := openai2.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// 创建日志器
	logger, _ := zap.NewProduction()
	sugarLogger := logger.Sugar()

	service := &EnterpriseAIService{
		config:         cfg,
		openaiClient:   client,
		limiter:        ratelimit.NewTokenBucketLimiter(100, 50), // 100 QPS, burst 50
		logger:         sugarLogger,
		semaphore:      make(chan struct{}, 100), // 最大并发 100
		maxConcurrency: 100,
		cache:          NewResponseCache(1000, 5*time.Minute),
		cacheEnabled:   true,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service, nil
}

// WithMaxConcurrency 设置最大并发
func WithMaxConcurrency(n int) EnterpriseAIServiceOption {
	return func(s *EnterpriseAIService) {
		s.maxConcurrency = n
		s.semaphore = make(chan struct{}, n)
	}
}

// WithRateLimit 设置限流
func WithRateLimit(rate, burst int) EnterpriseAIServiceOption {
	return func(s *EnterpriseAIService) {
		s.limiter = ratelimit.NewTokenBucketLimiter(rate, burst)
	}
}

// WithCache 设置缓存
func WithCache(enabled bool, maxSize int, ttl time.Duration) EnterpriseAIServiceOption {
	return func(s *EnterpriseAIService) {
		s.cacheEnabled = enabled
		s.cache = NewResponseCache(maxSize, ttl)
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages       []openai2.ChatMessage `json:"messages"`
	Model          string                `json:"model,omitempty"`
	Temperature    *float64              `json:"temperature,omitempty"`
	MaxTokens      *int                  `json:"max_tokens,omitempty"`
	TopP           *float64              `json:"top_p,omitempty"`
	ResponseFormat string                `json:"response_format,omitempty"`
	User           string                `json:"user,omitempty"`
	UseCache       bool                  `json:"use_cache,omitempty"`
}

// ChatResponse 聊天响应
//type ChatResponse struct {
//	ID           string                 `json:"id"`
//	Content      string                 `json:"content"`
//	Model        string                 `json:"model"`
//	Usage        *openai.Usage          `json:"usage,omitempty"`
//	FinishReason string                 `json:"finish_reason"`
//	Created      int64                  `json:"created"`
//	Metadata     map[string]interface{} `json:"metadata,omitempty"`
//	Cached       bool                   `json:"cached"`
//	LatencyMs    int64                  `json:"latency_ms"`
//	Chosice []
//}

// Chat 聊天
func (s *EnterpriseAIService) Chat(ctx context.Context, req *ChatRequest) (*openai2.ChatCompletionResponse, error) {
	startTime := time.Now()
	atomic.AddInt64(&s.requestCount, 1)

	// 检查缓存
	if s.cacheEnabled && req.UseCache {
		cacheKey := s.generateCacheKey(req)
		if cached, ok := s.cache.Get(cacheKey); ok {
			//cached.Cached = true
			//cached.LatencyMs = time.Since(startTime).Milliseconds()
			return cached, nil
		}
	}

	// 限流检查
	if err := s.limiter.Wait(ctx); err != nil {
		atomic.AddInt64(&s.failureCount, 1)
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// 并发控制
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		atomic.AddInt64(&s.failureCount, 1)
		return nil, ctx.Err()
	}

	// 构建请求
	chatReq := &openai2.ChatCompletionRequest{
		Model:    s.getModel(req.Model),
		Messages: req.Messages,
	}

	if req.Temperature != nil {
		chatReq.Temperature = req.Temperature
	}
	if req.MaxTokens != nil {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.TopP != nil {
		chatReq.TopP = req.TopP
	}
	if req.ResponseFormat != "" {
		chatReq.ResponseFormat = &openai2.ResponseFormat{Type: req.ResponseFormat}
	}
	if req.User != "" {
		chatReq.User = req.User
	}
	chatReq.Stream = false

	// 调用 API
	resp, err := s.openaiClient.ChatCompletion(ctx, chatReq)
	if err != nil {
		atomic.AddInt64(&s.failureCount, 1)
		s.logger.Errorw("chat completion failed", "error", err)
		return nil, err
	}

	atomic.AddInt64(&s.successCount, 1)
	latency := time.Since(startTime)
	atomic.AddInt64(&s.totalLatency, latency.Nanoseconds())

	// 构建响应
	//result := &ChatResponse{
	//	ID:           resp.ID,
	//	Content:      resp.Choices[0].Delta.Content,
	//	Model:        resp.Model,
	//	Usage:        &resp.Usage,
	//	FinishReason: resp.Choices[0].FinishReason,
	//	Created:      resp.Created,
	//	Cached:       false,
	//	LatencyMs:    latency.Milliseconds(),
	//}

	// 缓存结果
	if s.cacheEnabled && req.UseCache {
		cacheKey := s.generateCacheKey(req)
		s.cache.Set(cacheKey, resp)
	}

	return resp, nil
}

// StreamChat 流式聊天
func (s *EnterpriseAIService) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	atomic.AddInt64(&s.requestCount, 1)

	// 限流检查
	if err := s.limiter.Wait(ctx); err != nil {
		atomic.AddInt64(&s.failureCount, 1)
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// 并发控制
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		atomic.AddInt64(&s.failureCount, 1)
		return nil, ctx.Err()
	}

	// 构建请求
	chatReq := &openai2.ChatCompletionRequest{
		Model:    s.getModel(req.Model),
		Messages: req.Messages,
		Stream:   true,
	}

	if req.Temperature != nil {
		chatReq.Temperature = req.Temperature
	}
	if req.MaxTokens != nil {
		chatReq.MaxTokens = req.MaxTokens
	}

	// 调用流式 API
	chunkChan, err := s.openaiClient.StreamChatCompletion(ctx, chatReq)
	if err != nil {
		atomic.AddInt64(&s.failureCount, 1)
		return nil, err
	}

	// 转换为服务层流
	resultChan := make(chan StreamChunk, 100)

	go func() {
		defer close(resultChan)
		atomic.AddInt64(&s.successCount, 1)

		for chunk := range chunkChan {
			if chunk.Error != nil {
				resultChan <- StreamChunk{Error: chunk.Error}
				return
			}

			for _, choice := range chunk.Choices {
				resultChan <- StreamChunk{
					Content:      choice.Message.Content,
					FinishReason: choice.FinishReason,
				}
			}
		}
	}()

	return resultChan, nil
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
	Error        error  `json:"error,omitempty"`
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model,omitempty"`
	User  string   `json:"user,omitempty"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Object    string                  `json:"object"`
	Data      []openai2.EmbeddingData `json:"data"`
	Model     string                  `json:"model"`
	Usage     *openai2.Usage          `json:"usage"`
	LatencyMs int64                   `json:"latency_ms"`
}

// GetStats 获取统计信息
func (s *EnterpriseAIService) GetStats() ServiceStats {
	return ServiceStats{
		TotalRequests:  atomic.LoadInt64(&s.requestCount),
		SuccessCount:   atomic.LoadInt64(&s.successCount),
		FailureCount:   atomic.LoadInt64(&s.failureCount),
		TotalLatencyNs: atomic.LoadInt64(&s.totalLatency),
		ClientStats:    s.openaiClient.GetStats(),
	}
}

// ServiceStats 服务统计
type ServiceStats struct {
	TotalRequests  int64               `json:"total_requests"`
	SuccessCount   int64               `json:"success_count"`
	FailureCount   int64               `json:"failure_count"`
	TotalLatencyNs int64               `json:"total_latency_ns"`
	ClientStats    openai2.ClientStats `json:"client_stats"`
}

// getModel 获取模型名称
func (s *EnterpriseAIService) getModel(model string) string {
	if model != "" {
		return model
	}
	return s.config.OpenAIModelName
}

// generateCacheKey 生成缓存键
func (s *EnterpriseAIService) generateCacheKey(req *ChatRequest) string {
	return fmt.Sprintf("%s:%v:%v:%v", req.Model, req.Messages, req.Temperature, req.MaxTokens)
}

// Close 关闭服务
func (s *EnterpriseAIService) Close() error {
	return s.openaiClient.Close()
}

// ResponseCache 响应缓存
type ResponseCache struct {
	items   map[string]*cacheItem
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
}

type cacheItem struct {
	response  *openai2.ChatCompletionResponse
	expiresAt time.Time
}

// NewResponseCache 创建响应缓存
func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	cache := &ResponseCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// Get 获取缓存
func (c *ResponseCache) Get(key string) (*openai2.ChatCompletionResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.response, true
}

// Set 设置缓存
func (c *ResponseCache) Set(key string, response *openai2.ChatCompletionResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要清理
	if len(c.items) >= c.maxSize {
		c.evict()
	}

	c.items[key] = &cacheItem{
		response:  response,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// evict 淘汰缓存
func (c *ResponseCache) evict() {
	// 简单的 LRU：删除过期的或随机删除
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.expiresAt) {
			delete(c.items, k)
			return
		}
	}
	// 如果没有过期的，随机删除一个
	for k := range c.items {
		delete(c.items, k)
		return
	}
}

// cleanup 定期清理
func (c *ResponseCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
