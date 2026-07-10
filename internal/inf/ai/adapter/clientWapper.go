package adapter

import (
	"awesome/internal/inf/ai/config"
	openai2 "awesome/internal/inf/ai/openai"
	"context"
	"fmt"
	"github.com/ollama/ollama/api"
	"net/http"
	"net/url"
	"time"
)

type ClientWrapper struct {
	//client *Client
	ollama api.Client
}

func (w ClientWrapper) ChatCompletion(ctx context.Context, req *openai2.ChatCompletionRequest) (openai2.ChatCompletionResponse, error) {
	return openai2.ChatCompletionResponse{}, nil
}

func (w ClientWrapper) StreamChatCompletion(ctx context.Context, req *openai2.ChatCompletionRequest) (<-chan openai2.ChatCompletionResponse, error) {
	return nil, nil

}

func NewClientWrapper(cfg *config.AIConfig) (*ClientWrapper, error) {
	if cfg.ModelType == "ollama" {

		var httpClient *http.Client
		//if config.HTTPClient != nil {
		//	httpClient = config.HTTPClient
		//} else {
		//}
		httpClient = &http.Client{Timeout: time.Duration(cfg.Timeout)}

		baseURL, err := url.Parse(cfg.OpenAIBaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		cli := api.NewClient(baseURL, httpClient)
		return &ClientWrapper{ollama: *cli}, nil

	}
	return nil, fmt.Errorf("unsupported model type: %s", cfg.ModelType)

}
