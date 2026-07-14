package adapter

import (
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
	ollamaApi "github.com/eino-contrib/ollama/api"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"time"
)

type ChatAdapterImpl struct {
	chatModel      einoModel.ToolCallingChatModel
	modelName      string
	platformType   string
	baseURL        string // 用于创建临时 ollama 模型实例
	limiter        ratelimit.Limiter
	security       *security.Filter
	retryPolicy    *retry.Policy
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
		chatModel:    chatModel,
		limiter:      limiter,
		platformType: config.Type,
		modelName:    config.Model,
		baseURL:      config.BaseURL,
		retryPolicy: retry.NewPolicy(
			retry.WithMaxRetries(config.MaxRetries),
			retry.WithInitialDelay(100*time.Millisecond),
			retry.WithMaxDelay(5*time.Second),
			retry.WithMultiplier(2.0),
		),
		security: security.NewFilter(),
		metrics:  metrics.NewCollector(),
	}, nil

}

func (a *ChatAdapterImpl) Chat(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*model.ChatResponse, error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	opts := a.buildOptions(options)
	chatModel := a.getModelForRequest(options)

	result, err := chatModel.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("chat failed: %w", err)
	}

	return a.convertToResponse(result), nil
}

func (a *ChatAdapterImpl) ChatStream(ctx context.Context, messages []*schema.Message, options *ChatOptions) (*schema.StreamReader[*schema.Message], error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	opts := a.buildOptions(options)
	chatModel := a.getModelForRequest(options)

	streamChan, err := chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("chat stream failed: %w", err)
	}

	return streamChan, nil
}

// getModelForRequest 根据请求参数返回合适的 ChatModel 实例
// 对于 ollama 平台，如果请求指定了 thinking，则创建一个带 thinking 配置的临时实例
func (a *ChatAdapterImpl) getModelForRequest(options *ChatOptions) einoModel.ToolCallingChatModel {
	if a.platformType == "ollama" && options != nil && options.Thinking != nil {
		thinkValue := a.parseOllamaThinkValue(*options.Thinking)
		if thinkValue != nil {
			thinkModel, err := ollama.NewChatModel(context.Background(), &ollama.ChatModelConfig{
				BaseURL:  a.baseURL,
				Model:    a.modelName,
				Thinking: thinkValue,
			})
			if err == nil {
				return thinkModel
			}
		}
	}
	return a.chatModel
}

// parseOllamaThinkValue 将字符串 thinking 参数转换为 ollama 的 ThinkValue
// 支持: "true"/"false" 或 "high"/"medium"/"low"
func (a *ChatAdapterImpl) parseOllamaThinkValue(thinking string) *ollamaApi.ThinkValue {
	switch thinking {
	case "true", "enabled":
		return &ollamaApi.ThinkValue{Value: true}
	case "false", "disabled":
		return &ollamaApi.ThinkValue{Value: false}
	case "high", "medium", "low":
		return &ollamaApi.ThinkValue{Value: thinking}
	default:
		return nil
	}
}

func (a *ChatAdapterImpl) GetModelName() string {
	return a.modelName
}

func (a *ChatAdapterImpl) GetChatModel() einoModel.ToolCallingChatModel {
	return a.chatModel
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
		if options.TopP != nil {
			opts = append(opts, einoModel.WithTopP(float32(*options.TopP)))
		}
		if options.MaxTokens != nil {
			opts = append(opts, einoModel.WithMaxTokens(*options.MaxTokens))
		}
		if options.Stop != nil {
			opts = append(opts, einoModel.WithStop(*options.Stop))
		}
		if options.Tools != nil {
			opts = append(opts, einoModel.WithTools(options.Tools))
		}

		// 处理 thinking 参数
		if options.Thinking != nil {
			switch a.platformType {
			case "ark":
				thinkingType := arkModel.ThinkingType(*options.Thinking)
				opts = append(opts, ark.WithThinking(&arkModel.Thinking{
					Type: thinkingType,
				}))
			case "ollama":
				// ollama 的 thinking 通过 getModelForRequest 处理（创建带 thinking 配置的模型实例）
			}
		}
	}

	return opts
}

func (a *ChatAdapterImpl) convertToResponse(msg *schema.Message) *model.ChatResponse {
	// 转换 tool_calls
	var toolCalls []model.ToolCallInfo
	if len(msg.ToolCalls) > 0 {
		toolCalls = make([]model.ToolCallInfo, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			toolCalls[i] = model.ToolCallInfo{
				ID:   tc.ID,
				Type: tc.Type,
				Function: model.FunctionCallInfo{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	finishReason := ""
	if msg.ResponseMeta != nil {
		finishReason = msg.ResponseMeta.FinishReason
	}

	return &model.ChatResponse{
		ID:      a.platformType + "-" + generateID(),
		Object:  "chat.completion",
		Created: getCurrentTimestamp(),
		Model:   a.modelName,
		Choices: []model.Choice{
			{
				Index: 0,
				Message: model.Message{
					Role:      string(msg.Role),
					Content:   msg.Content,
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: model.Usage{
			PromptTokens:     msg.ResponseMeta.Usage.PromptTokens,
			CompletionTokens: msg.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      msg.ResponseMeta.Usage.TotalTokens,
		},
	}
}
