package rag

import (
	"awesome/internal/ai/rag/components"
	"awesome/internal/ai/rag/config"
	"awesome/internal/ai/rag/handlers"
	"awesome/internal/ai/rag/service"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"go.uber.org/dig"
)

// RegisterRAGServices 注册 RAG 相关服务到依赖注入容器
func RegisterRAGServices(c *dig.Container) error {
	// 注册 RAG 配置
	if err := c.Provide(func() *config.RAGConfig {
		return config.LoadRAGConfig()
	}); err != nil {
		return err
	}

	// 注册 Embedding 提供者
	if err := c.Provide(func(cfg *config.RAGConfig) *components.EmbeddingProvider {
		return components.NewEmbeddingProvider(cfg)
	}); err != nil {
		return err
	}

	// 注册 Embedding 模型
	if err := c.Provide(func(provider *components.EmbeddingProvider) (embedding.Embedder, error) {
		return provider.GetEmbeddingModel()
	}); err != nil {
		return err
	}

	// 注册向量存储提供者
	if err := c.Provide(func(cfg *config.RAGConfig, embedder embedding.Embedder) *components.VectorStoreProvider {
		return components.NewVectorStoreProvider(cfg, embedder)
	}); err != nil {
		return err
	}

	// 注册向量存储
	if err := c.Provide(func(provider *components.VectorStoreProvider) (components.VectorStore, error) {
		return provider.GetVectorStore()
	}); err != nil {
		return err
	}

	// 注册 RAG 服务
	if err := c.Provide(func(
		chatModel model.ChatModel,
		vectorStore components.VectorStore,
		cfg *config.RAGConfig,
	) service.RAGService {
		return service.NewRAGService(chatModel, vectorStore, cfg)
	}); err != nil {
		return err
	}

	// 注册 RAG 处理器
	if err := c.Provide(func(ragService service.RAGService) *handlers.RAGHandler {
		return handlers.NewRAGHandler(ragService)
	}); err != nil {
		return err
	}

	return nil
}
