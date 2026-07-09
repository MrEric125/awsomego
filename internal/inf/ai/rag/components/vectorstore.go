package components

import (
	"awesome/internal/inf/ai/rag/config"
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// VectorStore 向量存储接口
type VectorStore interface {
	// AddDocuments 添加文档
	AddDocuments(ctx context.Context, docs []*schema.Document) error

	// SimilaritySearch 相似度搜索
	SimilaritySearch(ctx context.Context, query string, topK int, opts ...retriever.Option) ([]*schema.Document, error)

	// Delete 删除文档
	Delete(ctx context.Context, ids []string) error

	// Clear 清空所有文档
	Clear(ctx context.Context) error
}

// MemoryVectorStore 内存向量存储实现
type MemoryVectorStore struct {
	embedder  embedding.Embedder
	documents []*schema.Document
	vectors   [][]float64
	mu        sync.RWMutex
	config    *config.RAGConfig
}

// NewMemoryVectorStore 创建内存向量存储
func NewMemoryVectorStore(embedder embedding.Embedder, cfg *config.RAGConfig) *MemoryVectorStore {
	return &MemoryVectorStore{
		embedder:  embedder,
		documents: make([]*schema.Document, 0),
		vectors:   make([][]float64, 0),
		config:    cfg,
	}
}

// AddDocuments 添加文档到向量存储
func (s *MemoryVectorStore) AddDocuments(ctx context.Context, docs []*schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// 提取文本内容
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// 生成文档向量
	vectors, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to embed documents: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 添加文档和向量
	s.documents = append(s.documents, docs...)
	s.vectors = append(s.vectors, vectors...)

	return nil
}

// SimilaritySearch 相似度搜索
func (s *MemoryVectorStore) SimilaritySearch(ctx context.Context, query string, topK int, opts ...retriever.Option) ([]*schema.Document, error) {
	// 生成查询向量
	queryVectors, err := s.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	queryVector := queryVectors[0]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return []*schema.Document{}, nil
	}

	// 计算相似度
	type scoredDoc struct {
		doc   *schema.Document
		score float64
	}

	scoredDocs := make([]scoredDoc, len(s.documents))
	for i, doc := range s.documents {
		score := cosineSimilarity(queryVector, s.vectors[i])
		scoredDocs[i] = scoredDoc{doc: doc, score: score}
	}

	// 按相似度排序
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// 返回 topK 结果
	resultCount := topK
	if resultCount > len(scoredDocs) {
		resultCount = len(scoredDocs)
	}

	results := make([]*schema.Document, resultCount)
	for i := 0; i < resultCount; i++ {
		// 将相似度分数添加到元数据
		doc := scoredDocs[i].doc
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		doc.MetaData["score"] = scoredDocs[i].score
		results[i] = doc
	}

	return results, nil
}

// Delete 删除文档
func (s *MemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	newDocs := make([]*schema.Document, 0)
	newVectors := make([][]float64, 0)

	for i, doc := range s.documents {
		if docID, ok := doc.MetaData["id"].(string); ok {
			if !idSet[docID] {
				newDocs = append(newDocs, doc)
				newVectors = append(newVectors, s.vectors[i])
			}
		}
	}

	s.documents = newDocs
	s.vectors = newVectors

	return nil
}

// Clear 清空所有文档
func (s *MemoryVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents = make([]*schema.Document, 0)
	s.vectors = make([][]float64, 0)

	return nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// VectorStoreProvider 向量存储提供者
type VectorStoreProvider struct {
	config   *config.RAGConfig
	embedder embedding.Embedder
}

// NewVectorStoreProvider 创建向量存储提供者
func NewVectorStoreProvider(cfg *config.RAGConfig, embedder embedding.Embedder) *VectorStoreProvider {
	return &VectorStoreProvider{
		config:   cfg,
		embedder: embedder,
	}
}

// GetVectorStore 获取向量存储
func (p *VectorStoreProvider) GetVectorStore() (VectorStore, error) {
	switch p.config.VectorStoreType {
	case "memory":
		return NewMemoryVectorStore(p.embedder, p.config), nil
	case "pinecone":
		// TODO: 实现 Pinecone 向量存储
		return nil, fmt.Errorf("pinecone vector store not implemented yet")
	case "milvus":
		return NewMilvusVectorStore(p.embedder, p.config)
	case "chroma":
		// TODO: 实现 Chroma 向量存储
		return nil, fmt.Errorf("chroma vector store not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", p.config.VectorStoreType)
	}
}
