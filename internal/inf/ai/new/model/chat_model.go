package model

import (
	"sync"
	"time"
)

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model" binding:"required"`
	Messages    []Message `json:"messages" binding:"required,min=1"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	Timeout     int       `json:"timeout,omitempty"` // 超时时间(秒)
}

// Message 消息结构
type Message struct {
	Role    string `json:"role" binding:"required,oneof=system user assistant"`
	Content string `json:"content" binding:"required"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"`
	Created   int64     `json:"created"`
	Model     string    `json:"model"`
	Choices   []Choice  `json:"choices"`
	Usage     Usage     `json:"usage"`
	Timestamp time.Time `json:"timestamp"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`
	Delta        Message `json:"delta,omitempty"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// Usage 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent SSE流事件
type StreamEvent struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Error   string   `json:"error,omitempty"`
}

// ConnectionPool 连接池
type ConnectionPool struct {
	maxSize     int
	connections map[int]*PooledConnection
	nextID      int32
	active      int32
	mu          sync.RWMutex
}

// PooledConnection 连接池中的连接
type PooledConnection struct {
	ID         int
	LastUsed   time.Time
	Active     bool
	CreateTime time.Time
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures         int32
	lastFailureTime  time.Time
	state            int32
	mu               sync.RWMutex
}
