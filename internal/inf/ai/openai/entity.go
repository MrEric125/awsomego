package openai

import (
	"awesome/internal/inf/ai/config"
	"awesome/internal/inf/ai/openai/audit"
	"awesome/internal/inf/ai/openai/metrics"
	"awesome/internal/inf/ai/openai/retry"
	"awesome/internal/inf/ai/openai/security"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

type T struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// PooledConnection 连接池中的连接
type PooledConnection struct {
	ID         int
	LastUsed   time.Time
	Active     bool
	CreateTime time.Time
}

// ConnectionPool 连接池
type ConnectionPool struct {
	maxSize     int
	connections map[int]*PooledConnection
	nextID      int32
	active      int32
	mu          sync.RWMutex
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
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatChoice           `json:"choices"`
	Usage             Usage                  `json:"usage"`
	SystemFingerprint string                 `json:"system_fingerprint"`
	Error             error                  `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

//
//type ChatCompletionResponse struct {
//	Model              string      `json:"model"`
//	CreatedAt          time.Time   `json:"created_at"`
//	Message            ChatMessage `json:"message"`
//	Done               bool        `json:"done"`
//	DoneReason         string      `json:"done_reason"`
//	TotalDuration      int64       `json:"total_duration"`
//	LoadDuration       int64       `json:"load_duration"`
//	PromptEvalCount    int         `json:"prompt_eval_count"`
//	PromptEvalDuration int         `json:"prompt_eval_duration"`
//	EvalCount          int         `json:"eval_count"`
//	EvalDuration       int         `json:"eval_duration"`
//}

// ChatChoice 聊天选择
type ChatChoice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason"`
	Finished     bool         `json:"finished"`
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

// ClientStats 客户端统计
type ClientStats struct {
	TotalRequests  int64
	SuccessCount   int64
	FailureCount   int64
	CircuitState   string
	ConnectionPool PoolStats
}

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

const (
	// StateClosed 关闭状态（正常）
	StateClosed CircuitState = iota
	// StateOpen 开启状态（熔断）
	StateOpen
	// StateHalfOpen 半开状态（尝试恢复）
	StateHalfOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures         int32
	lastFailureTime  time.Time
	state            int32
	mu               sync.RWMutex
}
