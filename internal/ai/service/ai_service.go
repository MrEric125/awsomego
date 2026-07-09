package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"awesome/internal/ai/components"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// AIService AI 服务接口
type AIService interface {
	// Chat 简单对话
	Chat(ctx context.Context, message string) (string, error)

	// ChatWithHistory 带历史记录的对话
	ChatWithHistory(ctx context.Context, messages []Message) (string, error)

	// StreamChat 流式对话（返回 channel）
	StreamChat(ctx context.Context, message string) (<-chan schema.Message, error)

	// Summarize 文本摘要
	Summarize(ctx context.Context, text string) (string, error)

	// Translate 翻译
	Translate(ctx context.Context, text, targetLang string) (string, error)
}

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIServiceImpl AI 服务实现
type AIServiceImpl struct {
	//eino 使用的模型
	chatModel model.BaseChatModel
}

// NewAIService 创建 AI 服务
func NewAIService(provider *components.ChatModelProvider) (AIService, error) {
	chatModel, err := provider.GetDefaultChatModel()
	if err != nil {
		return nil, fmt.Errorf("failed to get chat model: %w", err)
	}

	return &AIServiceImpl{
		chatModel: chatModel,
	}, nil
}

// Chat 简单对话
func (s *AIServiceImpl) Chat(ctx context.Context, message string) (string, error) {
	messages := []*schema.Message{
		schema.SystemMessage("你是一个有用的AI助手。"),
		schema.UserMessage(message),
	}

	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("chat generation failed: %w", err)
	}

	return resp.Content, nil
}

// ChatWithHistory 带历史记录的对话
func (s *AIServiceImpl) ChatWithHistory(ctx context.Context, messages []Message) (string, error) {
	var schemaMessages []*schema.Message

	for _, msg := range messages {
		switch strings.ToLower(msg.Role) {
		case "system":
			schemaMessages = append(schemaMessages, schema.SystemMessage(msg.Content))
		case "user":
			schemaMessages = append(schemaMessages, schema.UserMessage(msg.Content))
		case "assistant":
			schemaMessages = append(schemaMessages, schema.AssistantMessage(msg.Content, nil))
		default:
			return "", fmt.Errorf("unknown role: %s", msg.Role)
		}
	}

	resp, err := s.chatModel.Generate(ctx, schemaMessages)
	if err != nil {
		return "", fmt.Errorf("chat with history failed: %w", err)
	}

	return resp.Content, nil
}

// StreamChat 流式对话
func (s *AIServiceImpl) StreamChat(ctx context.Context, message string) (<-chan schema.Message, error) {
	messages := []*schema.Message{
		schema.SystemMessage("你是一个有用的AI助手。"),
		schema.UserMessage(message),
	}

	stream, err := s.chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("stream chat failed: %w", err)
	}

	ch := make(chan schema.Message, 10)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}

			// 只发送有内容的 chunk，避免空消息阻塞 channel
			if chunk.Content != "" {
				ch <- *schema.AssistantMessage(chunk.Content, nil)
			}
		}
	}()

	return ch, nil
}

// Summarize 文本摘要
func (s *AIServiceImpl) Summarize(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf("请对以下文本进行简洁的摘要（不超过100字）：\n\n%s", text)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的文本摘要助手。"),
		schema.UserMessage(prompt),
	}

	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	return resp.Content, nil
}

// Translate 翻译
func (s *AIServiceImpl) Translate(ctx context.Context, text, targetLang string) (string, error) {
	prompt := fmt.Sprintf("请将以下文本翻译成%s：\n\n%s", targetLang, text)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的翻译助手，能够准确翻译多种语言。"),
		schema.UserMessage(prompt),
	}

	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	return resp.Content, nil
}
