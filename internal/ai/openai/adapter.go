package openai

import (
	"context"
	"fmt"

	"awesome/internal/ai/config"
	"awesome/internal/ai/openai/ratelimit"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ChatModelAdapter OpenAI 聊天模型适配器
type ChatModelAdapter struct {
	client  *Client
	limiter ratelimit.Limiter
	config  *config.AIConfig
}

// NewChatModelAdapter 创建 OpenAI 聊天模型适配器
func NewChatModelAdapter(cfg *config.AIConfig) (*ChatModelAdapter, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// 创建限流器（每秒 100 请求，突发 20）
	limiter := ratelimit.NewTokenBucketLimiter(100, 20)

	return &ChatModelAdapter{
		client:  client,
		limiter: limiter,
		config:  cfg,
	}, nil
}

// Generate 生成响应
func (a *ChatModelAdapter) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 限流检查
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// 转换消息格式
	chatMessages := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 构建请求
	req := &ChatCompletionRequest{
		Model:    a.config.OpenAIModelName,
		Messages: chatMessages,
	}

	// 应用选项
	options := &model.Options{}
	model.GetCommonOptions(options, opts...)
	if options.Temperature != nil {
		temp := float64(*options.Temperature)
		req.Temperature = &temp
	}
	if options.MaxTokens != nil {
		maxTokens := int(*options.MaxTokens)
		req.MaxTokens = &maxTokens
	}
	if options.Model != nil {
		req.Model = *options.Model
	}

	// 调用 API
	resp, err := a.client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	//if len(resp.Message) == 0 {
	//	return nil, fmt.Errorf("no response choices returned")
	//}

	// 转换响应
	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp.Message.Content,
	}, nil
}

// Stream 流式生成
func (a *ChatModelAdapter) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// 限流检查
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// 转换消息格式
	chatMessages := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	// 构建请求
	req := &ChatCompletionRequest{
		Model:    a.config.OpenAIModelName,
		Messages: chatMessages,
		Stream:   false,
	}

	// 应用选项
	options := &model.Options{}
	model.GetCommonOptions(options, opts...)
	if options.Temperature != nil {
		temp := float64(*options.Temperature)
		req.Temperature = &temp
	}
	if options.MaxTokens != nil {
		maxTokens := int(*options.MaxTokens)
		req.MaxTokens = &maxTokens
	}
	if options.Model != nil {
		req.Model = *options.Model
	}

	// 调用流式 API
	chunkChan, err := a.client.StreamChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stream chat completion failed: %w", err)
	}

	// 创建流读取器
	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer sw.Close()
		fmt.Println("DEBUG: start reading chunks") // 确认 goroutine 启动了

		for chunk := range chunkChan {
			fmt.Printf("DEBUG: received chunk, choices=%d\n", len(chunk.Choices))
			if chunk.Error != nil {
				// 发送错误并关闭
				fmt.Println("DEBUG: chunk error:", chunk.Error) // 发送错误并关闭
			}

			for _, choice := range chunk.Choices {
				if choice.Message.Content != "" {
					msg := &schema.Message{
						Role:    schema.Assistant,
						Content: choice.Message.Content,
					}
					if sw.Send(msg, nil) {
						//return
					}
				}
			}
		}
	}()

	return sr, nil
}

// BindTools 绑定工具
func (a *ChatModelAdapter) BindTools(tools []*schema.ToolInfo) error {
	// OpenAI 支持工具调用
	return nil
}

// Close 关闭适配器
func (a *ChatModelAdapter) Close() error {
	return a.client.Close()
}

// GetClient 获取底层客户端
func (a *ChatModelAdapter) GetClient() *Client {
	return a.client
}

// GetMetrics 获取指标
func (a *ChatModelAdapter) GetMetrics() ClientStats {
	return a.client.GetStats()
}
