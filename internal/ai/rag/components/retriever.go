package components

import (
	"context"
	"fmt"

	"awesome/internal/ai/rag/config"

	"github.com/cloudwego/eino/schema"
)

// RAGRetriever RAG 检索器
type RAGRetriever struct {
	vectorStore VectorStore
	config      *config.RAGConfig
}

// NewRAGRetriever 创建 RAG 检索器
func NewRAGRetriever(vectorStore VectorStore, cfg *config.RAGConfig) *RAGRetriever {
	return &RAGRetriever{
		vectorStore: vectorStore,
		config:      cfg,
	}
}

// Retrieve 检索相关文档
func (r *RAGRetriever) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
	// 执行相似度搜索
	docs, err := r.vectorStore.SimilaritySearch(ctx, query, r.config.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 过滤低分文档
	if r.config.ScoreThreshold > 0 {
		filtered := make([]*schema.Document, 0)
		for _, doc := range docs {
			if score, ok := doc.MetaData["score"].(float64); ok {
				if score >= r.config.ScoreThreshold {
					filtered = append(filtered, doc)
				}
			} else {
				// 如果没有分数，保留文档
				filtered = append(filtered, doc)
			}
		}
		docs = filtered
	}

	return docs, nil
}

// HybridRetriever 混合检索器（结合关键词和向量检索）
type HybridRetriever struct {
	vectorStore   VectorStore
	config        *config.RAGConfig
	keywordWeight float64 // 关键词检索权重
	vectorWeight  float64 // 向量检索权重
}

// NewHybridRetriever 创建混合检索器
func NewHybridRetriever(vectorStore VectorStore, cfg *config.RAGConfig) *HybridRetriever {
	return &HybridRetriever{
		vectorStore:   vectorStore,
		config:        cfg,
		keywordWeight: 0.3,
		vectorWeight:  0.7,
	}
}

// Retrieve 混合检索
func (r *HybridRetriever) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
	// 向量检索
	vectorDocs, err := r.vectorStore.SimilaritySearch(ctx, query, r.config.TopK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// TODO: 添加关键词检索
	// 可以使用 BM25 或其他关键词检索算法

	// 目前只返回向量检索结果
	return vectorDocs, nil
}
