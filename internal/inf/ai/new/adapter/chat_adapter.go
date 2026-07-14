package adapter

import (
	"awesome/internal/inf/ai/new/model"
	"context"
	"fmt"
	einoModel "github.com/cloudwego/eino/components/model"

	"github.com/cloudwego/eino/schema"
)

// ChatAdapter 统一聊天适配器接口
type ChatAdapter interface {
	// Chat 普通对话
	Chat(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*model.ChatResponse, error)

	// ChatStream 流式对话
	ChatStream(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*schema.StreamReader[*schema.Message], error)

	// GetModelName 获取模型名称
	GetModelName() string

	GetChatModel() einoModel.ToolCallingChatModel

	// Close 关闭适配器
	Close() error
}

// ChatOptions 聊天选项
type ChatOptions struct {
	Temperature *float64
	MaxTokens   *int
	TopP        *float64
	Stop        *[]string
	Tools       []*schema.ToolInfo
	Thinking    *string // thinking模式: "enabled", "disabled", "auto"
}

// AdapterFactory 适配器工厂
type AdapterFactory struct {
	adapters map[string]ChatAdapter
}

// NewAdapterFactory 创建适配器工厂
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		adapters: make(map[string]ChatAdapter),
	}
}

// RegisterAdapter 注册适配器
func (f *AdapterFactory) RegisterAdapter(name string, adapter ChatAdapter) {
	f.adapters[name] = adapter
}

// GetAdapter 获取适配器
func (f *AdapterFactory) GetAdapter(name string) (ChatAdapter, error) {
	adapter, ok := f.adapters[name]
	if !ok {
		return nil, fmt.Errorf("adapter not found for model: %s", name)
	}
	return adapter, nil
}

// ListAdapters 列出所有适配器
func (f *AdapterFactory) ListAdapters() []string {
	names := make([]string, 0, len(f.adapters))
	for name := range f.adapters {
		names = append(names, name)
	}
	return names
}

// Close 关闭所有适配器
func (f *AdapterFactory) Close() error {
	for _, adapter := range f.adapters {
		if err := adapter.Close(); err != nil {
			return err
		}
	}
	return nil
}
