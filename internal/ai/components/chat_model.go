package components

import (
	"context"
	"fmt"
	"sync"

	"awesome/internal/ai/config"
	"awesome/internal/ai/openai"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
)

// ProviderType 提供者类型
type ProviderType string

const (
	ProviderArk    ProviderType = "ark"
	ProviderOpenAI ProviderType = "openai"
)

// ChatModelProvider 聊天模型提供者
type ChatModelProvider struct {
	config         *config.AIConfig
	openaiAdapter  *openai.ChatModelAdapter
	arkModel       model.ChatModel
	mu             sync.RWMutex
	defaultProvider ProviderType
}

// NewChatModelProvider 创建聊天模型提供者
func NewChatModelProvider(cfg *config.AIConfig) *ChatModelProvider {
	return &ChatModelProvider{
		config:         cfg,
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

	if p.config.ArkAPIKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY is not set")
	}

	temperature := float32(p.config.Temperature)
	topP := float32(p.config.TopP)
	maxTokens := p.config.MaxTokens

	chatModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		Model:       p.config.ArkModelName,
		APIKey:      p.config.ArkAPIKey,
		BaseURL:     p.config.ArkBaseURL,
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

	if p.config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	adapter, err := openai.NewChatModelAdapter(p.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI adapter: %w", err)
	}

	p.openaiAdapter = adapter
	return adapter, nil
}

// GetChatModel 获取指定类型的聊天模型
func (p *ChatModelProvider) GetChatModel(providerType ProviderType) (model.ChatModel, error) {
	switch providerType {
	case ProviderArk:
		return p.GetArkChatModel()
	case ProviderOpenAI:
		return p.GetOpenAIChatModel()
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// GetDefaultChatModel 获取默认聊天模型
func (p *ChatModelProvider) GetDefaultChatModel() (model.ChatModel, error) {
	// 根据配置选择默认提供者
	if p.config.ArkAPIKey != "" {
		return p.GetArkChatModel()
	}

	if p.config.OpenAIAPIKey != "" {
		return p.GetOpenAIChatModel()
	}

	return nil, fmt.Errorf("no available chat model provider")
}

// SetDefaultProvider 设置默认提供者
func (p *ChatModelProvider) SetDefaultProvider(provider ProviderType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultProvider = provider
}

// GetOpenAIClient 获取 OpenAI 客户端（用于高级功能）
func (p *ChatModelProvider) GetOpenAIClient() (*openai.Client, error) {
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
