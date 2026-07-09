package components

import (
	"awesome/internal/inf/ai/rag/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusVectorStore Milvus 向量存储实现
type MilvusVectorStore struct {
	client     client.Client
	embedder   embedding.Embedder
	config     *config.RAGConfig
	collection string
}

// NewMilvusVectorStore 创建 Milvus 向量存储
func NewMilvusVectorStore(embedder embedding.Embedder, cfg *config.RAGConfig) (*MilvusVectorStore, error) {
	// 构建 Milvus 连接地址
	milvusAddr := fmt.Sprintf("%s:%d", cfg.MilvusHost, cfg.MilvusPort)

	// 创建 Milvus 客户端
	c, err := client.NewClient(context.Background(), client.Config{
		Address: milvusAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Milvus client: %w", err)
	}

	store := &MilvusVectorStore{
		client:     c,
		embedder:   embedder,
		config:     cfg,
		collection: cfg.MilvusCollection,
	}

	// 确保集合存在
	if err := store.ensureCollection(context.Background()); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection 确保集合存在
func (s *MilvusVectorStore) ensureCollection(ctx context.Context) error {
	// 检查集合是否存在
	has, err := s.client.HasCollection(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !has {
		// 创建集合
		schema := &entity.Schema{
			CollectionName: s.collection,
			Description:    "RAG knowledge base",
			AutoID:         true,
			Fields: []*entity.Field{
				{
					Name:       "id",
					DataType:   entity.FieldTypeInt64,
					PrimaryKey: true,
					AutoID:     true,
				},
				{
					Name:     "content",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						"max_length": "65535",
					},
				},
				{
					Name:     "metadata",
					DataType: entity.FieldTypeJSON,
				},
				{
					Name:     "embedding",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						"dim": fmt.Sprintf("%d", s.config.VectorDimension),
					},
				},
				{
					Name:     "created_at",
					DataType: entity.FieldTypeInt64,
				},
			},
		}

		if err := s.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		// 创建索引
		idx, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}

		if err := s.client.CreateIndex(ctx, s.collection, "embedding", idx, false); err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}
	}

	// 加载集合到内存
	if err := s.client.LoadCollection(ctx, s.collection, false); err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	return nil
}

// AddDocuments 添加文档到 Milvus
func (s *MilvusVectorStore) AddDocuments(ctx context.Context, docs []*schema.Document) error {
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

	// 准备插入数据
	contents := make([]string, len(docs))
	metadatas := make([][]byte, len(docs))
	embeddings := make([][]float32, len(docs))
	createdAts := make([]int64, len(docs))

	now := time.Now().Unix()

	for i, doc := range docs {
		contents[i] = doc.Content

		// 序列化元数据
		if doc.MetaData != nil {
			metaBytes, err := json.Marshal(doc.MetaData)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadatas[i] = metaBytes
		} else {
			metadatas[i] = []byte("{}")
		}

		// 转换向量为 float32
		embeddings[i] = make([]float32, len(vectors[i]))
		for j, v := range vectors[i] {
			embeddings[i][j] = float32(v)
		}

		createdAts[i] = now
	}

	// 插入数据
	columns := []entity.Column{
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnFloatVector("embedding", s.config.VectorDimension, embeddings),
		entity.NewColumnInt64("created_at", createdAts),
	}

	if _, err := s.client.Insert(ctx, s.collection, "", columns...); err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	// 刷新以确保数据持久化
	if err := s.client.Flush(ctx, s.collection, false); err != nil {
		return fmt.Errorf("failed to flush collection: %w", err)
	}

	return nil
}

// SimilaritySearch 相似度搜索
func (s *MilvusVectorStore) SimilaritySearch(ctx context.Context, query string, topK int, opts ...retriever.Option) ([]*schema.Document, error) {
	// 生成查询向量
	queryVectors, err := s.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 转换为 float32
	queryVector := make([]float32, len(queryVectors[0]))
	for i, v := range queryVectors[0] {
		queryVector[i] = float32(v)
	}

	// 执行搜索
	sp, err := entity.NewIndexAUTOINDEXSearchParam(10)
	if err != nil {
		return nil, fmt.Errorf("failed to create search param: %w", err)
	}

	searchResult, err := s.client.Search(
		ctx,
		s.collection,
		[]string{},                      // partitions
		"",                              // expr
		[]string{"content", "metadata"}, // output fields
		[]entity.Vector{entity.FloatVector(queryVector)},
		"embedding",
		entity.L2,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 解析结果
	var results []*schema.Document

	if len(searchResult) > 0 {
		for _, result := range searchResult[0].Fields {
			// 获取内容列
			if contentCol, ok := result.(*entity.ColumnVarChar); ok {
				for i, content := range contentCol.Data() {
					doc := &schema.Document{
						Content:  content,
						MetaData: make(map[string]any),
					}

					// 添加相似度分数
					if i < len(searchResult[0].Scores) {
						// Milvus 返回的是距离，需要转换为相似度分数
						distance := searchResult[0].Scores[i]
						// 使用 1 / (1 + distance) 转换为 0-1 之间的分数
						score := 1.0 / (1.0 + distance)
						doc.MetaData["score"] = score
					}

					results = append(results, doc)
				}
			}
		}

		// 获取元数据列
		for _, result := range searchResult[0].Fields {
			if metaCol, ok := result.(*entity.ColumnJSONBytes); ok {
				for i, metaBytes := range metaCol.Data() {
					if i < len(results) {
						var metadata map[string]any
						if err := json.Unmarshal(metaBytes, &metadata); err == nil {
							for k, v := range metadata {
								results[i].MetaData[k] = v
							}
						}
					}
				}
			}
		}
	}

	return results, nil
}

// Delete 删除文档
func (s *MilvusVectorStore) Delete(ctx context.Context, ids []string) error {
	// Milvus 需要通过表达式删除
	// 这里我们使用 id 字段进行删除
	// 注意：需要先查询出对应的 Milvus 内部 ID

	// 由于我们的 schema 使用 AutoID，外部 ID 存储在 metadata 中
	// 这里简化处理，通过 metadata.id 进行过滤删除
	expr := fmt.Sprintf(`metadata["id"] in [%s]`, joinStringIDs(ids))

	if err := s.client.Delete(ctx, s.collection, "", expr); err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// Clear 清空所有文档
func (s *MilvusVectorStore) Clear(ctx context.Context) error {
	// 删除集合
	if err := s.client.DropCollection(ctx, s.collection); err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	// 重新创建集合
	if err := s.ensureCollection(ctx); err != nil {
		return fmt.Errorf("failed to recreate collection: %w", err)
	}

	return nil
}

// Close 关闭 Milvus 连接
func (s *MilvusVectorStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// joinStringIDs 连接字符串 ID
func joinStringIDs(ids []string) string {
	result := ""
	for i, id := range ids {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, id)
	}
	return result
}
