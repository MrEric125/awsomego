package service

import (
	"awesome/internal/inf/ai/new/adapter"
	"awesome/internal/inf/ai/new/config"
	"awesome/internal/inf/ai/new/model"
	"awesome/internal/inf/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/eino-contrib/jsonschema"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type ChatService struct {
	factory *adapter.AdapterFactory
	config  *config.Config
}

func NewChatService(cfg config.Config) (*ChatService, error) {
	factory := adapter.NewAdapterFactory()

	// 注册所有适配器
	for name, modelCfg := range cfg.Models {

		chatAdapter, err := adapter.NewChatAdapter(modelCfg)

		if err != nil {
			return nil, err
		}

		factory.RegisterAdapter(name, chatAdapter)
		//logger.Info("registered model adapter", zap.String("model", name))
	}

	return &ChatService{
		factory: factory,
		config:  &cfg,
		//logger:  logger,
	}, nil
}

func (s *ChatService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	startTime := time.Now()
	logger.Logger.Info("processing chat request",
		zap.String("model", req.Model),
		zap.Int("messages_count", len(req.Messages)),
		zap.Bool("stream", req.Stream))

	// 获取适配器
	chatAdapter, err := s.factory.GetAdapter(req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get adapter: %w", err)
	}

	// 转换消息格式
	messages := s.convertMessages(req.Messages)

	// 构建选项
	options := &adapter.ChatOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Thinking:    req.Thinking,
		Tools:       s.convertTools(req.Tools),
	}

	// 设置超时
	timeout := s.config.Timeout.Default
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
		if timeout > s.config.Timeout.Max {
			timeout = s.config.Timeout.Max
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 调用模型
	response, err := chatAdapter.Chat(ctx, messages, options)
	if err != nil {
		logger.Logger.Error("chat failed",
			zap.String("model", req.Model),
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))
		return nil, fmt.Errorf("chat failed: %w", err)
	}

	response.Timestamp = time.Now()

	logger.Logger.Info("chat completed",
		zap.String("model", req.Model),
		zap.Duration("duration", time.Since(startTime)))

	return response, nil
}

func (s *ChatService) ChatStream(ctx context.Context, req *model.ChatRequest) (*schema.StreamReader[*schema.Message], error) {
	logger.Logger.Info("processing stream chat request",
		zap.String("model", req.Model),
		zap.Int("messages_count", len(req.Messages)))

	// 获取适配器
	chatAdapter, err := s.factory.GetAdapter(req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get adapter: %w", err)
	}

	// 转换消息格式
	messages := s.convertMessages(req.Messages)

	// 构建选项
	options := &adapter.ChatOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Thinking:    req.Thinking,
		Tools:       s.convertTools(req.Tools),
	}

	// 调用流式模型
	streamChan, err := chatAdapter.ChatStream(ctx, messages, options)
	if err != nil {
		logger.Logger.Error("stream chat failed",
			zap.String("model", req.Model),
			zap.Error(err))
		return nil, fmt.Errorf("stream chat failed: %w", err)
	}

	return streamChan, nil
}

func (s *ChatService) ListModels() []string {
	return s.factory.ListAdapters()
}

func (s *ChatService) convertMessages(messages []model.Message) []*schema.Message {
	result := make([]*schema.Message, len(messages))
	for i, msg := range messages {
		m := &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		}
		// 处理 tool 角色的消息
		if msg.Role == "tool" {
			m.ToolCallID = msg.ToolCallID
			m.ToolName = msg.Name
		}
		result[i] = m
	}
	return result
}

// convertTools 将 API 请求中的工具定义转换为 eino 的 schema.ToolInfo
func (s *ChatService) convertTools(tools []model.ToolDefinition) []*schema.ToolInfo {
	if len(tools) == 0 {
		return nil
	}

	result := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		toolInfo := &schema.ToolInfo{
			Name: t.Function.Name,
			Desc: t.Function.Description,
		}

		// 将 JSON Schema 格式的参数转换为 eino 的 ParamsOneOf
		if t.Function.Parameters != nil {
			paramsJSON, err := json.Marshal(t.Function.Parameters)
			if err == nil {
				js := &jsonschema.Schema{}
				if err := json.Unmarshal(paramsJSON, js); err == nil {
					toolInfo.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(js)
				}
			}
		}

		result = append(result, toolInfo)
	}
	return result
}

func (s *ChatService) Close() error {
	return s.factory.Close()
}
