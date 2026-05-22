package config

import (
	"fmt"
	"os"
)

// AIConfig AI 服务配置
type AIConfig struct {
	// 豆包 Ark 配置
	ArkAPIKey    string
	ArkModelName string
	ArkBaseURL   string

	// OpenAI 配置（可选）
	OpenAIAPIKey    string
	OpenAIModelName string
	OpenAIBaseURL   string

	// 通用配置
	Timeout     int // 超时时间（秒）
	MaxRetries  int // 最大重试次数
	Temperature float64
	TopP        float64
	MaxTokens   int
}

// LoadAIConfig 从环境变量加载 AI 配置
func LoadAIConfig() *AIConfig {
	return &AIConfig{
		// 豆包 Ark 配置
		ArkAPIKey:    getEnv("ARK_API_KEY", ""),
		ArkModelName: getEnv("ARK_MODEL_NAME", ""),
		ArkBaseURL:   getEnv("ARK_BASE_URL", ""),

		// OpenAI 配置
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		OpenAIModelName: getEnv("OPENAI_MODEL_NAME", ""),
		OpenAIBaseURL:   getEnv("OPENAI_BASE_URL", ""),

		// 通用配置
		Timeout:     getIntEnv("AI_TIMEOUT", 30),
		MaxRetries:  getIntEnv("AI_MAX_RETRIES", 3),
		Temperature: getFloatEnv("AI_TEMPERATURE", 0.7),
		TopP:        getFloatEnv("AI_TOP_P", 0.9),
		MaxTokens:   getIntEnv("AI_MAX_TOKENS", 2000),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var result float64
		if _, err := fmt.Sscanf(value, "%f", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
