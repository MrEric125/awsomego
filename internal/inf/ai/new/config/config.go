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
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "release"),
		},
		Models: map[string]ModelConfig{
			"openai-gpt4": {
				Type:    "openai",
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
			},
			"openai-gpt3": {
				Type:    "openai",
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-3.5-turbo",
			},
			"deepseek": {
				Type:    "openai",
				APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
				BaseURL: "https://api.deepseek.com/v1",
				Model:   "deepseek-chat",
			},
			"ollama-llama2": {
				Type:    "ollama",
				BaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
				Model:   "llama2",
			},
			"ark-doubao": {
				Type:    "ark",
				APIKey:  os.Getenv("ARK_API_KEY"),
				BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
				Model:   "doubao-pro-32k",
			},
		},
		Timeout: TimeoutConfig{
			Default: 30 * time.Second,
			Max:     120 * time.Second,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
