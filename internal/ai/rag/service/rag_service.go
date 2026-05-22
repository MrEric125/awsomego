package service

import (
	"context"
	"fmt"
	"strings"

	"awesome/internal/ai/rag/components"
	"awesome/internal/ai/rag/config"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RAGService RAG 服务接口
type RAGService interface {
	// AddDocuments 添加文档到知识库
	AddDocuments(ctx context.Context, docs []*Document) error

	// AddText 添加文本到知识库
	AddText(ctx context.Context, text string, metadata map[string]any) error

	// Query RAG 查询
	Query(ctx context.Context, question string) (*RAGResponse, error)

	// QueryWithHistory 带历史记录的 RAG 查询
	QueryWithHistory(ctx context.Context, question string, history []Message) (*RAGResponse, error)

	// StreamQuery 流式 RAG 查询
	StreamQuery(ctx context.Context, question string) (<-chan string, error)

	// Clear 清空知识库
	Clear(ctx context.Context) error
}

// Document 文档结构
type Document struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RAGResponse RAG 响应
type RAGResponse struct {
	Answer     string   `json:"answer"`
	Sources    []Source `json:"sources,omitempty"`
	Confidence float64  `json:"confidence"`
}

// Source 来源文档
type Source struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Score    float64        `json:"score"`
}

// RAGServiceImpl RAG 服务实现
type RAGServiceImpl struct {
	chatModel   model.ChatModel
	retriever   *components.RAGRetriever
	vectorStore components.VectorStore
	processor   *components.DocumentProcessor
	config      *config.RAGConfig
}

// NewRAGService 创建 RAG 服务
func NewRAGService(
	chatModel model.ChatModel,
	vectorStore components.VectorStore,
	cfg *config.RAGConfig,
) RAGService {
	retriever := components.NewRAGRetriever(vectorStore, cfg)
	processor := components.NewDocumentProcessor(cfg)

	return &RAGServiceImpl{
		chatModel:   chatModel,
		retriever:   retriever,
		vectorStore: vectorStore,
		processor:   processor,
		config:      cfg,
	}
}

// AddDocuments 添加文档到知识库
func (s *RAGServiceImpl) AddDocuments(ctx context.Context, docs []*Document) error {
	// 转换为 schema.Document
	schemaDocs := make([]*schema.Document, len(docs))
	for i, doc := range docs {
		schemaDocs[i] = &schema.Document{
			Content: doc.Content,
			MetaData:    doc.Metadata,
		}
	}

	// 分割文档
	splitDocs := s.processor.SplitDocuments(schemaDocs)

	// 添加到向量存储
	if err := s.vectorStore.AddDocuments(ctx, splitDocs); err != nil {
		return fmt.Errorf("failed to add documents to vector store: %w", err)
	}

	return nil
}

// AddText 添加文本到知识库
func (s *RAGServiceImpl) AddText(ctx context.Context, text string, metadata map[string]any) error {
	docs := []*Document{
		{
			Content:  text,
			Metadata: metadata,
		},
	}
	return s.AddDocuments(ctx, docs)
}

// Query RAG 查询
func (s *RAGServiceImpl) Query(ctx context.Context, question string) (*RAGResponse, error) {
	// 1. 检索相关文档
	docs, err := s.retriever.Retrieve(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 2. 构建上下文
	context := s.buildContext(docs)

	// 3. 生成回答
	answer, err := s.generateAnswer(ctx, question, context)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// 4. 构建响应
	response := &RAGResponse{
		Answer:  answer,
		Sources: s.buildSources(docs),
	}

	// 计算置信度
	if len(docs) > 0 {
		if score, ok := docs[0].MetaData["score"].(float64); ok {
			response.Confidence = score
		}
	}

	return response, nil
}

// QueryWithHistory 带历史记录的 RAG 查询
func (s *RAGServiceImpl) QueryWithHistory(ctx context.Context, question string, history []Message) (*RAGResponse, error) {
	// 1. 检索相关文档
	docs, err := s.retriever.Retrieve(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 2. 构建上下文
	context := s.buildContext(docs)

	// 3. 生成带历史的回答
	answer, err := s.generateAnswerWithHistory(ctx, question, context, history)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// 4. 构建响应
	response := &RAGResponse{
		Answer:  answer,
		Sources: s.buildSources(docs),
	}

	if len(docs) > 0 {
		if score, ok := docs[0].MetaData["score"].(float64); ok {
			response.Confidence = score
		}
	}

	return response, nil
}

// StreamQuery 流式 RAG 查询
func (s *RAGServiceImpl) StreamQuery(ctx context.Context, question string) (<-chan string, error) {
	// 1. 检索相关文档
	docs, err := s.retriever.Retrieve(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 2. 构建上下文
	context := s.buildContext(docs)

	// 3. 流式生成回答
	return s.streamGenerateAnswer(ctx, question, context)
}

// Clear 清空知识库
func (s *RAGServiceImpl) Clear(ctx context.Context) error {
	return s.vectorStore.Clear(ctx)
}

// buildContext 构建上下文
func (s *RAGServiceImpl) buildContext(docs []*schema.Document) string {
	var contexts []string
	for i, doc := range docs {
		contexts = append(contexts, fmt.Sprintf("[文档%d]\n%s", i+1, doc.Content))
	}
	return strings.Join(contexts, "\n\n")
}

// buildSources 构建来源列表
func (s *RAGServiceImpl) buildSources(docs []*schema.Document) []Source {
	sources := make([]Source, len(docs))
	for i, doc := range docs {
		source := Source{
			Content:  doc.Content,
			Metadata: doc.MetaData,
		}
		if score, ok := doc.MetaData["score"].(float64); ok {
			source.Score = score
		}
		sources[i] = source
	}
	return sources
}

// generateAnswer 生成回答
func (s *RAGServiceImpl) generateAnswer(ctx context.Context, question, context string) (string, error) {
	prompt := fmt.Sprintf(`你是一个专业的问答助手。请根据提供的上下文信息回答用户的问题。

上下文信息：
%s

用户问题：%s

请基于上下文信息给出准确、详细的回答。如果上下文中没有相关信息，请明确说明。`, context, question)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的问答助手，擅长根据提供的上下文信息回答问题。"),
		schema.UserMessage(prompt),
	}

	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// generateAnswerWithHistory 带历史记录生成回答
func (s *RAGServiceImpl) generateAnswerWithHistory(ctx context.Context, question, context string, history []Message) (string, error) {
	prompt := fmt.Sprintf(`你是一个专业的问答助手。请根据提供的上下文信息回答用户的问题。

上下文信息：
%s

用户问题：%s

请基于上下文信息给出准确、详细的回答。如果上下文中没有相关信息，请明确说明。`, context, question)

	// 构建消息列表
	messages := make([]*schema.Message, 0)
	messages = append(messages, schema.SystemMessage("你是一个专业的问答助手，擅长根据提供的上下文信息回答问题。"))

	// 添加历史消息
	for _, msg := range history {
		switch strings.ToLower(msg.Role) {
		case "user":
			messages = append(messages, schema.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(msg.Content, nil))
		}
	}

	// 添加当前问题
	messages = append(messages, schema.UserMessage(prompt))

	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// streamGenerateAnswer 流式生成回答
func (s *RAGServiceImpl) streamGenerateAnswer(ctx context.Context, question, context string) (<-chan string, error) {
	prompt := fmt.Sprintf(`你是一个专业的问答助手。请根据提供的上下文信息回答用户的问题。

上下文信息：
%s

用户问题：%s

请基于上下文信息给出准确、详细的回答。如果上下文中没有相关信息，请明确说明。`, context, question)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的问答助手，擅长根据提供的上下文信息回答问题。"),
		schema.UserMessage(prompt),
	}

	stream, err := s.chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 10)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			chunk, err := stream.Recv()
			if err != nil {
				return
			}

			if chunk.Content != "" {
				ch <- chunk.Content
			}

			// 检查是否结束
			if len(chunk.Content) == 0 {
				break
			}
		}
	}()

	return ch, nil
}
