package adapter

import (
	"awesome/internal/inf/ai/config"
	"awesome/internal/inf/ai/openai"
	"awesome/internal/inf/ai/openai/ratelimit"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ChatModelAdapter OpenAI 聊天模型适配器
type ChatModelAdapter struct {
	client   *openai.Client
	provider string
	limiter  ratelimit.Limiter
	config   *config.AIConfig
}

// NewChatModelAdapter 创建 OpenAI 聊天模型适配器
func NewChatModelAdapter(cfg *config.AIConfig, provider string) (*ChatModelAdapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "openai"
	}

	cfgCopy := *cfg
	switch provider {
	case "openai":
		if cfgCopy.OpenAIBaseURL == "" {
			return nil, fmt.Errorf("OPENAI_BASE_URL is required for openai provider")
		}
	case "deepseek":
		if cfgCopy.DeepSeekURL == "" {
			return nil, fmt.Errorf("DEEPSEEK_URL is required for deepseek provider")
		}
		cfgCopy.OpenAIBaseURL = cfgCopy.DeepSeekURL
		cfgCopy.OpenAIAPIKey = cfgCopy.DeepSeekAPIKey
		if cfgCopy.OpenAIModelName == "" {
			cfgCopy.OpenAIModelName = cfgCopy.DeepSeekModel
		}
	case "ollama":
		if cfgCopy.OllamaBaseURL == "" {
			return nil, fmt.Errorf("OLLAMA_BASE_URL is required for ollama provider")
		}
		cfgCopy.OpenAIBaseURL = cfgCopy.OllamaBaseURL
		cfgCopy.OpenAIAPIKey = ""
		if cfgCopy.OpenAIModelName == "" {
			cfgCopy.OpenAIModelName = cfgCopy.OllamaModel
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	client, err := openai.NewClient(&cfgCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for provider %s: %w", provider, err)
	}

	limiter := ratelimit.NewTokenBucketLimiter(100, 20)

	return &ChatModelAdapter{
		client:   client,
		provider: provider,
		limiter:  limiter,
		config:   &cfgCopy,
	}, nil
}

// Generate 生成响应
func (a *ChatModelAdapter) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	chatMessages := make([]openai.ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = openai.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := &openai.ChatCompletionRequest{
		Model:    a.config.OpenAIModelName,
		Messages: chatMessages,
	}

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
	if options.TopP != nil {
		topP := float64(*options.TopP)
		req.TopP = &topP
	}

	resp, err := a.client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp.Choices[0].Message.Content,
	}, nil
}

func (a *ChatModelAdapter) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	chatMessages := make([]openai.ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = openai.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := &openai.ChatCompletionRequest{
		Model:    a.config.OpenAIModelName,
		Messages: chatMessages,
		Stream:   true,
	}

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
	if options.TopP != nil {
		topP := float64(*options.TopP)
		req.TopP = &topP
	}

	chunkChan, err := a.client.StreamChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stream chat completion failed: %w", err)
	}

	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer sw.Close()

		for chunk := range chunkChan {
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			if choice.Delta == nil {
				continue
			}
			if choice.Delta.Content == "" {
				continue
			}

			_ = sw.Send(&schema.Message{Content: choice.Delta.Content}, nil)
		}
	}()

	return sr, nil
}

// Provider returns the provider name used by this adapter.
func (a *ChatModelAdapter) Provider() string {
	return a.provider
}

// BindTools implements model.ChatModel.
func (a *ChatModelAdapter) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

// GetClient returns the underlying OpenAI-compatible client.
func (a *ChatModelAdapter) GetClient() *openai.Client {
	return a.client
}

// Close closes adapter resources.
func (a *ChatModelAdapter) Close() error {
	a.client.Close()
	return nil
}

func (a *ChatModelAdapter) GetMetrics() openai.ClientStats {
	return a.client.GetStats()
}

