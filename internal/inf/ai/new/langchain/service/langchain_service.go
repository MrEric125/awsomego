package service

import (
	"awesome/internal/inf/ai/new/langchain/components"
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
)

// LangChainService LangChain 服务接口
type LangChainService interface {
	// Chat 简单对话
	Chat(ctx context.Context, message string) (string, error)

	// ChatWithProvider 使用指定提供商对话
	ChatWithProvider(ctx context.Context, message, provider string) (string, error)

	// ChatWithHistory 带历史记录的对话
	ChatWithHistory(ctx context.Context, messages []Message) (string, error)

	// RunChain 运行链
	RunChain(ctx context.Context, prompt string, chainType ChainType) (string, error)

	// Summarize 文本摘要
	Summarize(ctx context.Context, text string) (string, error)

	// Translate 翻译
	Translate(ctx context.Context, text, targetLang string) (string, error)

	// QuestionAnswering 问答（基于文档）
	QuestionAnswering(ctx context.Context, question string, documents []string) (string, error)

	// ListProviders 列出可用的提供商
	ListProviders() []string
}

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChainType 链类型
type ChainType string

const (
	// SimpleChain 简单链
	SimpleChain ChainType = "simple"
	// ConversationalChain 对话链
	ConversationalChain ChainType = "conversational"
	// SummarizationChain 摘要链
	SummarizationChain ChainType = "summarization"
)

// LangChainServiceImpl LangChain 服务实现
type LangChainServiceImpl struct {
	llm      llms.Model
	memory   *memory.ConversationBuffer
	provider *components.LLMProvider
}

// NewLangChainService 创建 LangChain 服务
func NewLangChainService(provider *components.LLMProvider) (LangChainService, error) {
	llm, err := provider.GetDefaultLLM()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	return &LangChainServiceImpl{
		llm:      llm,
		memory:   memory.NewConversationBuffer(),
		provider: provider,
	}, nil
}

// ChatWithProvider 使用指定提供商对话
func (s *LangChainServiceImpl) ChatWithProvider(ctx context.Context, message, provider string) (string, error) {
	llm, err := s.provider.GetLLMByProvider(provider)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM for provider %s: %w", provider, err)
	}

	chain := chains.NewLLMChain(
		llm,
		prompts.NewPromptTemplate(
			"You are a helpful AI assistant. {{.input}}",
			[]string{"input"},
		),
	)

	result, err := chains.Predict(ctx, chain, map[string]any{
		"input": message,
	})
	if err != nil {
		return "", fmt.Errorf("chat with provider %s failed: %w", provider, err)
	}

	return result, nil
}

// Chat 简单对话
func (s *LangChainServiceImpl) Chat(ctx context.Context, message string) (string, error) {
	// 使用 LLMChain
	chain := chains.NewLLMChain(
		s.llm,
		prompts.NewPromptTemplate(
			"You are a helpful AI assistant. {{.input}}",
			[]string{"input"},
		),
	)

	result, err := chains.Predict(ctx, chain, map[string]any{
		"input": message,
	})
	if err != nil {
		return "", fmt.Errorf("chat failed: %w", err)
	}

	return result, nil
}

// ChatWithHistory 带历史记录的对话
func (s *LangChainServiceImpl) ChatWithHistory(ctx context.Context, messages []Message) (string, error) {
	// 创建对话链
	chain := chains.NewConversation(s.llm, s.memory)

	// 添加历史消息到内存
	for _, msg := range messages {
		switch strings.ToLower(msg.Role) {
		case "user":
			s.memory.ChatHistory.AddUserMessage(ctx, msg.Content)
		case "assistant":
			s.memory.ChatHistory.AddAIMessage(ctx, msg.Content)
		case "system":
			// System messages are handled differently
			continue
		}
	}

	// 获取最后一条用户消息
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.ToLower(messages[i].Role) == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	if lastUserMessage == "" {
		return "", fmt.Errorf("no user message found")
	}

	result, err := chains.Predict(ctx, chain, map[string]any{
		"input": lastUserMessage,
	})
	if err != nil {
		return "", fmt.Errorf("chat with history failed: %w", err)
	}

	return result, nil
}

// RunChain 运行链
func (s *LangChainServiceImpl) RunChain(ctx context.Context, prompt string, chainType ChainType) (string, error) {
	switch chainType {
	case SimpleChain:
		chain := chains.NewLLMChain(
			s.llm,
			prompts.NewPromptTemplate(prompt, []string{"input"}),
		)
		return chains.Predict(ctx, chain, map[string]any{"input": prompt})

	case ConversationalChain:
		chain := chains.NewConversation(s.llm, s.memory)
		return chains.Predict(ctx, chain, map[string]any{"input": prompt})

	case SummarizationChain:
		chain := chains.LoadMapReduceSummarization(s.llm)
		docs := []schema.Document{{PageContent: prompt}}
		return chains.Run(ctx, chain, docs)

	default:
		return "", fmt.Errorf("unknown chain type: %s", chainType)
	}
}

// Summarize 文本摘要
func (s *LangChainServiceImpl) Summarize(ctx context.Context, text string) (string, error) {
	chain := chains.LoadStuffSummarization(s.llm)
	docs := []schema.Document{{PageContent: text}}

	result, err := chains.Run(ctx, chain, docs)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	return result, nil
}

// Translate 翻译
func (s *LangChainServiceImpl) Translate(ctx context.Context, text, targetLang string) (string, error) {
	prompt := fmt.Sprintf("Translate the following text to %s:\n\n%s", targetLang, text)

	chain := chains.NewLLMChain(
		s.llm,
		prompts.NewPromptTemplate("{{.input}}", []string{"input"}),
	)

	result, err := chains.Predict(ctx, chain, map[string]any{
		"input": prompt,
	})
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	return result, nil
}

// QuestionAnswering 问答(基于文档)
func (s *LangChainServiceImpl) QuestionAnswering(ctx context.Context, question string, documents []string) (string, error) {
	// 将文档转换为 schema.Document
	docs := make([]schema.Document, len(documents))
	for i, doc := range documents {
		docs[i] = schema.Document{PageContent: doc}
	}

	// 使用 Stuff Documents Chain
	chain := chains.LoadStuffQA(s.llm)

	result, err := chains.Run(ctx, chain, docs, chains.WithMaxLength(1000))
	if err != nil {
		return "", fmt.Errorf("question answering failed: %w", err)
	}

	return result, nil
}

// ListProviders 列出可用的提供商
func (s *LangChainServiceImpl) ListProviders() []string {
	providers := []string{}

	// 检查哪些提供商已配置
	if s.provider.Config().OpenAIAPIKey != "" {
		providers = append(providers, "openai")
	}
	if s.provider.Config().AzureAPIKey != "" && s.provider.Config().AzureEndpoint != "" {
		providers = append(providers, "azure")
	}
	if s.provider.Config().AnthropicAPIKey != "" {
		providers = append(providers, "anthropic")
	}
	if s.provider.Config().GoogleAPIKey != "" {
		providers = append(providers, "google")
	}
	if s.provider.Config().DeepSeekAPIKey != "" {
		providers = append(providers, "deepseek")
	}
	// Ollama 始终可用（本地）
	providers = append(providers, "ollama")

	return providers
}
