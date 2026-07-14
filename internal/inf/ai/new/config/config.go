package config

import (
	"os"
	"time"
)

type Config struct {
	Server  ServerConfig
	Models  map[string]ModelConfig
	Timeout TimeoutConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type ModelConfig struct {
	Type       string                 `json:"type"` // openai, ollama, ark
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url"`
	Model      string                 `json:"model"`
	Options    map[string]interface{} `json:"options"`
	MaxRetries int                    `json:"maxRetries"` // 最大重试次数
}

type TimeoutConfig struct {
	Default time.Duration
	Max     time.Duration
}

func LoadConfig() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "release"),
		},
		Models: map[string]ModelConfig{},
		Timeout: TimeoutConfig{
			Default: 30 * time.Second,
			Max:     120 * time.Second,
		},
	}
	if ollamaModel := getEnv("OLLAMA_MODEL_NAME", ""); ollamaModel != "" {
		cfg.Models[ollamaModel] = ModelConfig{
			Type:    "ollama",
			BaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			Model:   ollamaModel,
		}
	}
	cfg.Models["deepseek-coder:6.7b"] = ModelConfig{
		Type:    "ollama",
		APIKey:  getEnv("DEEPSEEK_API_KEY", ""),
		BaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		Model:   "deepseek-coder:6.7b",
	}
	if kimiModel := getEnv("KIMI_MODEL_NAME", ""); kimiModel != "" {
		cfg.Models[kimiModel] = ModelConfig{
			Type:    "openai",
			APIKey:  getEnv("KIMI_API_KEY", ""),
			BaseURL: getEnv("KIMI_BASE_URL", "https://api.moonshot.cn/v1"),
			Model:   kimiModel,
		}
	}
	return cfg

}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
