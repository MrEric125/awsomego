package adapter

import (
	"awesome/internal/inf/ai/new/audit"
	"awesome/internal/inf/ai/new/config"
	"awesome/internal/inf/ai/new/metrics"
	"awesome/internal/inf/ai/new/model"
	"awesome/internal/inf/ai/new/ratelimit"
	"awesome/internal/inf/ai/new/retry"
	"awesome/internal/inf/ai/new/security"
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"time"
)

type ChatAdapterImpl struct {
	chatModel      einoModel.ToolCallingChatModel
	modelName      string
	limiter        ratelimit.Limiter
	security       *security.Filter
	retryPolicy    *retry.Policy
	audit          *audit.Logger
	metrics        *metrics.Collector
	connectionPool model.ConnectionPool
	circuitBreaker model.CircuitBreaker
}

func NewChatAdapter(config config.ModelConfig) (chatAdapter ChatAdapter, err error) {
	var chatModel einoModel.ToolCallingChatModel

	switch config.Type {
	case "openai":
		chatModel, err = openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			BaseURL: config.BaseURL,
			Model:   config.Model,
		})
	case "ollama":
		chatModel, err = ollama.NewChatModel(context.Background(), &ollama.ChatModelConfig{
			BaseURL: config.BaseURL,
			Model:   config.Model,
		})
	case "ark":
		chatModel, err = ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			BaseURL: config.BaseURL,
			Model:   config.Model,
		})
	default:
		return nil, fmt.Errorf("未找到模型")

	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %s chat model: %w", config.Type, err)
	}
	limiter := ratelimit.NewTokenBucketLimiter(10, 20)

	return &ChatAdapterImpl{
		chatModel: chatModel,
		limiter:   limiter,
		modelName: config.Model,
		retryPolicy: retry.NewPolicy(
			retry.WithMaxRetries(config.MaxRetries),
			retry.WithInitialDelay(100*time.Millisecond),
			retry.WithMaxDelay(5*time.Second),
			retry.WithMultiplier(2.0),
		),
		security: security.NewFilter(),
		//audit:          audit.NewLogger(sugarLogger),
		metrics: metrics.NewCollector(),
		//logger:         sugarLogger,
	}, nil

}

func (a *ChatAdapterImpl) Chat(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*model.ChatResponse, error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	opts := a.buildOptions(options)

	result, err := a.chatModel.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("Ollama chat failed: %w", err)
	}

	return a.convertToResponse(result), nil
}

func (a *ChatAdapterImpl) ChatStream(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*schema.StreamReader[*schema.Message], error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	opts := a.buildOptions(options)

	streamChan, err := a.chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("Ollama chat stream failed: %w", err)
	}

	return streamChan, nil
}

func (a *ChatAdapterImpl) GetModelName() string {
	return a.modelName
}

func (a *ChatAdapterImpl) Close() error {
	return nil
}

func (a *ChatAdapterImpl) buildOptions(options *ChatOptions) []einoModel.Option {
	var opts []einoModel.Option

	if options != nil {
		if options.Temperature != nil {
			opts = append(opts, einoModel.WithTemperature(float32(*options.Temperature)))
		}
	}

	return opts
}

func (a *ChatAdapterImpl) convertToResponse(msg *schema.Message) *model.ChatResponse {
	return &model.ChatResponse{
		ID:      "ollama-" + generateID(),
		Object:  "chat.completion",
		Created: getCurrentTimestamp(),
		Model:   a.modelName,
		Choices: []model.Choice{
			{
				Index: 0,
				Message: model.Message{
					Role:    string(msg.Role),
					Content: msg.Content,
				},
				FinishReason: "stop",
			},
		},
		Usage: model.Usage{},
	}
}

type OllamaConfig struct {
	BaseURL string
	Model   string
}
