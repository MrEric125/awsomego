package components

import (
	"context"
	"fmt"

	"awesome/internal/langchain/config"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// LLMProvider LangChain LLM 提供者
type LLMProvider struct {
	config *config.LangChainConfig
}

// NewLLMProvider 创建 LLM 提供者
func NewLLMProvider(cfg *config.LangChainConfig) *LLMProvider {
	return &LLMProvider{
		config: cfg,
	}
}

// Config 获取配置
func (p *LLMProvider) Config() *config.LangChainConfig {
	return p.config
}

// GetOpenAILLM 获取 OpenAI LLM
func (p *LLMProvider) GetOpenAILLM() (llms.Model, error) {
	if p.config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	opts := []openai.Option{
		openai.WithToken(p.config.OpenAIAPIKey),
		openai.WithModel(p.config.OpenAIModelName),
	}

	// 如果配置了自定义 BaseURL(如兼容 OpenAI 的 API)
	if p.config.OpenAIBaseURL != "" {
		opts = append(opts, openai.WithBaseURL(p.config.OpenAIBaseURL))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	return llm, nil
}

// GetOllamaLLM 获取 Ollama LLM(本地模型)
func (p *LLMProvider) GetOllamaLLM() (llms.Model, error) {
	opts := []ollama.Option{
		ollama.WithModel(p.config.OllamaModel),
		ollama.WithServerURL(p.config.OllamaBaseURL),
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama LLM: %w", err)
	}

	return llm, nil
}

// GetAzureOpenAILLM 获取 Azure OpenAI LLM
func (p *LLMProvider) GetAzureOpenAILLM() (llms.Model, error) {
	if p.config.AzureAPIKey == "" || p.config.AzureEndpoint == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_API_KEY or AZURE_OPENAI_ENDPOINT is not set")
	}

	opts := []openai.Option{
		openai.WithToken(p.config.AzureAPIKey),
		openai.WithModel(p.config.AzureDeployment),
		openai.WithBaseURL(fmt.Sprintf("%s/openai/deployments/%s", p.config.AzureEndpoint, p.config.AzureDeployment)),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure OpenAI LLM: %w", err)
	}

	return llm, nil
}

// GetAnthropicLLM 获取 Anthropic Claude LLM
func (p *LLMProvider) GetAnthropicLLM() (llms.Model, error) {
	if p.config.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	opts := []anthropic.Option{
		anthropic.WithToken(p.config.AnthropicAPIKey),
		anthropic.WithModel(p.config.AnthropicModel),
	}

	llm, err := anthropic.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic LLM: %w", err)
	}

	return llm, nil
}

// GetGoogleLLM 获取 Google Gemini LLM
func (p *LLMProvider) GetGoogleLLM() (llms.Model, error) {
	if p.config.GoogleAPIKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is not set")
	}

	opts := []googleai.Option{
		googleai.WithAPIKey(p.config.GoogleAPIKey),
		googleai.WithDefaultModel(p.config.GoogleModel),
	}

	llm, err := googleai.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Gemini LLM: %w", err)
	}

	return llm, nil
}

// GetDeepSeekLLM 获取 DeepSeek LLM（兼容 OpenAI API）
func (p *LLMProvider) GetDeepSeekLLM() (llms.Model, error) {
	if p.config.DeepSeekAPIKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set")
	}

	opts := []openai.Option{
		openai.WithToken(p.config.DeepSeekAPIKey),
		openai.WithModel(p.config.DeepSeekModel),
		openai.WithBaseURL(p.config.DeepSeekURL),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek LLM: %w", err)
	}

	return llm, nil
}

// GetDefaultLLM 获取默认 LLM（根据配置选择）
func (p *LLMProvider) GetDefaultLLM() (llms.Model, error) {
	switch p.config.DefaultProvider {
	case "azure":
		if p.config.AzureAPIKey != "" {
			return p.GetAzureOpenAILLM()
		}
	case "anthropic":
		if p.config.AnthropicAPIKey != "" {
			return p.GetAnthropicLLM()
		}
	case "google":
		if p.config.GoogleAPIKey != "" {
			return p.GetGoogleLLM()
		}
	case "ollama":
		return p.GetOllamaLLM()
	case "deepseek":
		if p.config.DeepSeekAPIKey != "" {
			return p.GetDeepSeekLLM()
		}
	case "openai":
		if p.config.OpenAIAPIKey != "" {
			return p.GetOpenAILLM()
		}
	}

	// 如果默认提供商不可用，按优先级尝试其他提供商
	if p.config.OpenAIAPIKey != "" {
		return p.GetOpenAILLM()
	}
	if p.config.AzureAPIKey != "" {
		return p.GetAzureOpenAILLM()
	}
	if p.config.AnthropicAPIKey != "" {
		return p.GetAnthropicLLM()
	}
	if p.config.GoogleAPIKey != "" {
		return p.GetGoogleLLM()
	}
	if p.config.DeepSeekAPIKey != "" {
		return p.GetDeepSeekLLM()
	}
	return p.GetOllamaLLM()
}

// GetLLMByProvider 根据提供商名称获取 LLM
func (p *LLMProvider) GetLLMByProvider(provider string) (llms.Model, error) {
	switch provider {
	case "openai":
		return p.GetOpenAILLM()
	case "azure":
		return p.GetAzureOpenAILLM()
	case "anthropic":
		return p.GetAnthropicLLM()
	case "google":
		return p.GetGoogleLLM()
	case "ollama":
		return p.GetOllamaLLM()
	case "deepseek":
		return p.GetDeepSeekLLM()
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// GenerateContent 使用指定 LLM 生成内容
func (p *LLMProvider) GenerateContent(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	llm, err := p.GetDefaultLLM()
	if err != nil {
		return "", err
	}

	// 添加默认的 temperature 和 top_p 选项
	defaultOpts := []llms.CallOption{
		llms.WithTemperature(p.config.Temperature),
		llms.WithTopP(p.config.TopP),
		llms.WithMaxTokens(p.config.MaxTokens),
	}

	// 合并用户传入的选项
	allOpts := append(defaultOpts, opts...)

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt, allOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return completion, nil
}
