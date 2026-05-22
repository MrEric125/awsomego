package config

import (
	"fmt"
	"os"
)

// RAGConfig RAG 服务配置
type RAGConfig struct {
	// Embedding 配置
	EmbeddingModel   string // embedding 模型名称
	EmbeddingAPIKey  string // embedding API Key
	EmbeddingBaseURL string // embedding API Base URL

	// 向量数据库配置
	VectorStoreType string // 向量存储类型: memory, pinecone, milvus, chroma
	VectorDimension int    // 向量维度

	// Pinecone 配置（可选）
	PineconeAPIKey      string
	PineconeEnvironment string
	PineconeIndexName   string

	// Milvus 配置（可选）
	MilvusHost       string
	MilvusPort       int
	MilvusCollection string

	// 检索配置
	TopK           int     // 检索返回的文档数量
	ScoreThreshold float64 // 相似度阈值

	// 文档分割配置
	ChunkSize    int // 文档块大小
	ChunkOverlap int // 文档块重叠大小
}

// LoadRAGConfig 从环境变量加载 RAG 配置
func LoadRAGConfig() *RAGConfig {
	return &RAGConfig{
		// Embedding 配置
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "text-embedding-ada-002"),
		EmbeddingAPIKey:  getEnv("EMBEDDING_API_KEY", ""),
		EmbeddingBaseURL: getEnv("EMBEDDING_BASE_URL", "https://api.openai.com/v1"),

		// 向量数据库配置
		VectorStoreType: getEnv("VECTOR_STORE_TYPE", "memory"),
		VectorDimension: getIntEnv("VECTOR_DIMENSION", 1536),

		// Pinecone 配置
		PineconeAPIKey:      getEnv("PINECONE_API_KEY", ""),
		PineconeEnvironment: getEnv("PINECONE_ENVIRONMENT", ""),
		PineconeIndexName:   getEnv("PINECONE_INDEX_NAME", ""),

		// Milvus 配置
		MilvusHost:       getEnv("MILVUS_HOST", "localhost"),
		MilvusPort:       getIntEnv("MILVUS_PORT", 19530),
		MilvusCollection: getEnv("MILVUS_COLLECTION", "documents"),

		// 检索配置
		TopK:           getIntEnv("RAG_TOP_K", 5),
		ScoreThreshold: getFloatEnv("RAG_SCORE_THRESHOLD", 0.7),

		// 文档分割配置
		ChunkSize:    getIntEnv("CHUNK_SIZE", 1000),
		ChunkOverlap: getIntEnv("CHUNK_OVERLAP", 200),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var result float64
		if _, err := fmt.Sscanf(value, "%f", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
