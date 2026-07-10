package components

import (
	"awesome/internal/inf/ai/adapter"
	"awesome/internal/inf/ai/config"
	openai2 "awesome/internal/inf/ai/openai"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
)

// ProviderType 提供者类型
type ProviderType string

const (
	ProviderArk      ProviderType = "ark"
	ProviderOpenAI   ProviderType = "openai"
	ProviderOllama   ProviderType = "ollama"
	ProviderDeepSeek ProviderType = "deepseek"
)

// ChatModelProvider 聊天模型提供者
type ChatModelProvider struct {
	config          *config.AIConfig
	openaiAdapter   *adapter.ChatModelAdapter
	arkModel        model.ChatModel
	mu              sync.RWMutex
	defaultProvider ProviderType
}

// NewChatModelProvider 创建聊天模型提供者
func NewChatModelProvider(cfg *config.AIConfig) *ChatModelProvider {
	return &ChatModelProvider{
		config:          cfg,
		defaultProvider: ProviderArk, // 默认使用 Ark
	}
}

// GetArkChatModel 获取豆包 Ark 聊天模型
func (p *ChatModelProvider) GetArkChatModel() (model.ChatModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果已创建，直接返回
	if p.arkModel != nil {
		return p.arkModel, nil
	}

	temperature := float32(p.config.Temperature)
	topP := float32(p.config.TopP)
	maxTokens := p.config.MaxTokens

	chatModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		Model:       p.config.OpenAIModelName,
		APIKey:      p.config.OpenAIAPIKey,
		BaseURL:     p.config.OpenAIBaseURL,
		Temperature: &temperature,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Ark chat model: %w", err)
	}

	p.arkModel = chatModel
	return chatModel, nil
}

// GetOpenAIChatModel 获取 OpenAI 聊天模型
func (p *ChatModelProvider) GetOpenAIChatModel() (model.ChatModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果已创建，直接返回
	if p.openaiAdapter != nil {
		return p.openaiAdapter, nil
	}

	adapter, err := adapter.NewChatModelAdapter(p.config, "openai")
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI adapter: %w", err)
	}

	p.openaiAdapter = adapter
	return adapter, nil
}

func (p *ChatModelProvider) GetOllamaChatModel() (model.ChatModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.openaiAdapter != nil && strings.ToLower(p.openaiAdapter.Provider()) == "ollama" {
		return p.openaiAdapter, nil
	}

	adapter, err := adapter.NewChatModelAdapter(p.config, "ollama")
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama adapter: %w", err)
	}

	p.openaiAdapter = adapter
	return adapter, nil
}

func (p *ChatModelProvider) GetDeepSeekChatModel() (model.ChatModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.openaiAdapter != nil && strings.ToLower(p.openaiAdapter.Provider()) == "deepseek" {
		return p.openaiAdapter, nil
	}

	adapter, err := adapter.NewChatModelAdapter(p.config, "deepseek")
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek adapter: %w", err)
	}

	p.openaiAdapter = adapter
	return adapter, nil
}

// GetChatModel 获取指定类型的聊天模型
func (p *ChatModelProvider) GetChatModel(provider string) (model.ChatModel, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "default":
		return p.GetDefaultChatModel()
	case "ark":
		return p.GetArkChatModel()
	case "openai":
		return p.GetOpenAIChatModel()
	case "ollama":
		return p.GetOllamaChatModel()
	case "deepseek":
		return p.GetDeepSeekChatModel()
	default:
		return nil, fmt.Errorf("unknown provider type: %s", provider)
	}
}

// GetDefaultChatModel 获取默认聊天模型
func (p *ChatModelProvider) GetDefaultChatModel() (model.ChatModel, error) {
	providerType := strings.ToLower(strings.TrimSpace(p.config.ModelType))

	switch providerType {
	case "ark":
		return p.GetArkChatModel()
	case "openai":
		return p.GetOpenAIChatModel()
	case "ollama":
		return p.GetOllamaChatModel()
	case "deepseek":
		return p.GetDeepSeekChatModel()
	}

	// 逐个尝试可用提供者
	if p.config.OpenAIAPIKey != "" && p.config.OpenAIBaseURL != "" {
		return p.GetOpenAIChatModel()
	}
	if p.config.OllamaBaseURL != "" {
		return p.GetOllamaChatModel()
	}
	if p.config.DeepSeekAPIKey != "" && p.config.DeepSeekURL != "" {
		return p.GetDeepSeekChatModel()
	}
	return p.GetArkChatModel()
}

// SetDefaultProvider 设置默认提供者
func (p *ChatModelProvider) SetDefaultProvider(provider ProviderType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultProvider = provider
}

// GetOpenAIClient 获取 OpenAI 客户端（用于高级功能）
func (p *ChatModelProvider) GetOpenAIClient() (*openai2.Client, error) {
	if p.openaiAdapter == nil {
		_, err := p.GetOpenAIChatModel()
		if err != nil {
			return nil, err
		}
	}
	return p.openaiAdapter.GetClient(), nil
}

// Close 关闭所有模型
func (p *ChatModelProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	if p.openaiAdapter != nil {
		if err := p.openaiAdapter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}

	return nil
}

func (p *ChatModelProvider) getOllamaChatModel() (model.ChatModel, error) {
	return p.GetOllamaChatModel()
}
