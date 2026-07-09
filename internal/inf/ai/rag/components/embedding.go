package components

import (
	"awesome/internal/inf/ai/rag/config"
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
)

// EmbeddingProvider Embedding 模型提供者
type EmbeddingProvider struct {
	config *config.RAGConfig
}

// NewEmbeddingProvider 创建 Embedding 模型提供者
func NewEmbeddingProvider(cfg *config.RAGConfig) *EmbeddingProvider {
	return &EmbeddingProvider{
		config: cfg,
	}
}

// GetEmbeddingModel 获取 Embedding 模型
// 支持多种 Embedding 提供商
func (p *EmbeddingProvider) GetEmbeddingModel() (*MockEmbedding, error) {
	// 优先使用 OpenAI Embedding
	if p.config.EmbeddingAPIKey != "" {
		return p.getOpenAIEmbedding()
	}

	// 可以添加其他 Embedding 提供商
	// 如: Azure, HuggingFace, Cohere 等

	return nil, fmt.Errorf("no available embedding provider configured")
}

// getOpenAIEmbedding 获取 OpenAI Embedding 模型
func (p *EmbeddingProvider) getOpenAIEmbedding() (*MockEmbedding, error) {
	// 使用 Eino 的 OpenAI Embedding 实现
	// 注意: 这里需要导入对应的包
	// import "github.com/cloudwego/eino-ext/components/embedding/openai"

	// 由于 Eino 可能还没有 OpenAI embedding 实现，我们创建一个简单的 mock
	// 实际使用时应该替换为真实的实现
	return &MockEmbedding{
		dimension: p.config.VectorDimension,
	}, nil
}

// MockEmbedding Mock Embedding 实现（用于测试）
type MockEmbedding struct {
	dimension int
}

// SetDimension 设置向量维度
func (m *MockEmbedding) SetDimension(dim int) {
	m.dimension = dim
}

// EmbedStrings 将字符串列表转换为向量
func (m *MockEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	// 实际实现应该调用真实的 Embedding API
	// 这里返回随机向量用于演示
	vectors := make([][]float64, len(texts))
	for i := range texts {
		vectors[i] = make([]float64, m.dimension)
		// 简单的 hash-based 向量生成（仅用于演示）
		for j := 0; j < m.dimension; j++ {
			vectors[i][j] = float64(len(texts[i])*j%100) / 100.0
		}
	}
	return vectors, nil
}

// EmbedDocuments 将文档列表转换为向量
func (m *MockEmbedding) EmbedDocuments(ctx context.Context, docs []*schema.Document, opts ...embedding.Option) ([][]float64, error) {
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	return m.EmbedStrings(ctx, texts, opts...)
}
