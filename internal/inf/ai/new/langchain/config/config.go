package config

import (
	"os"
	"strconv"
)

// LangChainConfig LangChain 配置结构
type LangChainConfig struct {
	// OpenAI 配置
	OpenAIAPIKey    string
	OpenAIModelName string
	OpenAIBaseURL   string

	// Azure OpenAI 配置
	AzureAPIKey     string
	AzureEndpoint   string
	AzureDeployment string
	AzureAPIVersion string

	// Anthropic Claude 配置
	AnthropicAPIKey string
	AnthropicModel  string

	// Google Gemini 配置
	GoogleAPIKey string
	GoogleModel  string

	// Ollama 配置（本地模型）
	OllamaBaseURL string
	OllamaModel   string

	// DeepSeek 配置（兼容 OpenAI API）
	DeepSeekAPIKey string
	DeepSeekModel  string
	DeepSeekURL    string

	// 通用配置
	Temperature float64
	TopP        float64
	MaxTokens   int
	Timeout     int
	MaxRetries  int

	// 默认提供商优先级
	DefaultProvider string // openai, azure, anthropic, google, ollama, deepseek
}

// LoadLangChainConfig 从环境变量加载 LangChain 配置
func LoadLangChainConfig() *LangChainConfig {
	temp, _ := strconv.ParseFloat(getEnv("LC_TEMPERATURE", "0.7"), 64)
	topP, _ := strconv.ParseFloat(getEnv("LC_TOP_P", "0.9"), 64)
	maxTokens, _ := strconv.Atoi(getEnv("LC_MAX_TOKENS", "2000"))
	timeout, _ := strconv.Atoi(getEnv("LC_TIMEOUT", "30"))
	maxRetries, _ := strconv.Atoi(getEnv("LC_MAX_RETRIES", "3"))

	return &LangChainConfig{
		// OpenAI
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		OpenAIModelName: getEnv("OPENAI_MODEL_NAME", "gpt-3.5-turbo"),
		OpenAIBaseURL:   getEnv("OPENAI_BASE_URL", ""),

		// Azure OpenAI
		AzureAPIKey:     getEnv("AZURE_OPENAI_API_KEY", ""),
		AzureEndpoint:   getEnv("AZURE_OPENAI_ENDPOINT", ""),
		AzureDeployment: getEnv("AZURE_OPENAI_DEPLOYMENT", "gpt-35-turbo"),
		AzureAPIVersion: getEnv("AZURE_OPENAI_API_VERSION", "2023-05-15"),

		// Anthropic Claude
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:  getEnv("ANTHROPIC_MODEL", "claude-3-sonnet-20240229"),

		// Google Gemini
		GoogleAPIKey: getEnv("GOOGLE_API_KEY", ""),
		GoogleModel:  getEnv("GOOGLE_MODEL", "gemini-pro"),

		// Ollama
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "llama2"),

		// DeepSeek
		DeepSeekAPIKey: getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekModel:  getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekURL:    getEnv("DEEPSEEK_URL", "https://api.deepseek.com/v1"),

		// 通用配置
		Temperature:     temp,
		TopP:            topP,
		MaxTokens:       maxTokens,
		Timeout:         timeout,
		MaxRetries:      maxRetries,
		DefaultProvider: getEnv("LC_DEFAULT_PROVIDER", "openai"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
