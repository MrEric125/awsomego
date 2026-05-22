package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"awesome/internal/ai/rag/components"
	"awesome/internal/ai/rag/config"
	"awesome/internal/ai/rag/service"

	"github.com/cloudwego/eino/schema"
)

func TestRagTest(t *testing.T) {
	fmt.Println("=== RAG 示例 ===")

	// 1. 加载配置
	cfg := config.LoadRAGConfig()
	fmt.Printf("RAG 配置: VectorStoreType=%s, TopK=%d, ChunkSize=%d\n",
		cfg.VectorStoreType, cfg.TopK, cfg.ChunkSize)

	// 2. 创建 Embedding 提供者
	embeddingProvider := components.NewEmbeddingProvider(cfg)
	embedder, err := embeddingProvider.GetEmbeddingModel()
	if err != nil {
		log.Fatalf("Failed to get embedding model: %v", err)
	}
	fmt.Println("Embedding 模型创建成功")

	// 3. 创建向量存储
	vectorStoreProvider := components.NewVectorStoreProvider(cfg, embedder)
	vectorStore, err := vectorStoreProvider.GetVectorStore()
	if err != nil {
		log.Fatalf("Failed to get vector store: %v", err)
	}
	fmt.Println("向量存储创建成功")

	// 4. 添加文档到知识库
	docs := []*schema.Document{
		{
			Content:  "Go 语言是由 Google 开发的开源编程语言。Go 语言具有简洁、高效、并发性强等特点。",
			MetaData: map[string]any{"source": "go_intro", "category": "programming"},
		},
		{
			Content:  "RAG（Retrieval-Augmented Generation）是一种结合检索和生成的 AI 技术。它通过检索相关文档来增强生成模型的回答质量。",
			MetaData: map[string]any{"source": "rag_intro", "category": "ai"},
		},
		{
			Content:  "向量数据库是专门用于存储和检索向量数据的数据库。常见的向量数据库包括 Pinecone、Milvus、Chroma 等。",
			MetaData: map[string]any{"source": "vector_db", "category": "database"},
		},
	}

	// 分割文档
	processor := components.NewDocumentProcessor(cfg)
	splitDocs := processor.SplitDocuments(docs)

	// 添加到向量存储
	ctx := context.Background()
	if err := vectorStore.AddDocuments(ctx, splitDocs); err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("成功添加 %d 个文档块到知识库\n", len(splitDocs))

	// 5. 创建检索器
	retriever := components.NewRAGRetriever(vectorStore, cfg)

	// 6. 执行检索
	queries := []string{
		"什么是 Go 语言？",
		"RAG 是什么？",
		"有哪些向量数据库？",
	}

	for _, query := range queries {
		fmt.Printf("\n查询: %s\n", query)
		results, err := retriever.Retrieve(ctx, query)
		if err != nil {
			log.Printf("检索失败: %v", err)
			continue
		}

		fmt.Printf("找到 %d 个相关文档:\n", len(results))
		for i, doc := range results {
			score := 0.0
			if s, ok := doc.MetaData["score"].(float64); ok {
				score = s
			}
			fmt.Printf("  [%d] 相似度: %.4f\n", i+1, score)
			fmt.Printf("      内容: %s\n", truncate(doc.Content, 100))
		}
	}

	// 7. 演示 RAG 服务（需要 ChatModel）
	fmt.Println("\n=== RAG 服务演示 ===")
	fmt.Println("注意: 完整的 RAG 服务需要配置 ChatModel（如豆包 Ark 或 OpenAI）")
	fmt.Println("请设置 ARK_API_KEY 或 OPENAI_API_KEY 环境变量")

	// 演示服务接口
	var ragSvc service.RAGService = service.NewRAGService(nil, vectorStore, cfg)

	// 添加文档
	fmt.Println("\n添加测试文档...")
	if err := ragSvc.AddText(ctx, "这是一个测试文档，用于演示 RAG 功能。", nil); err != nil {
		log.Printf("添加文档失败: %v", err)
	} else {
		fmt.Println("文档添加成功")
	}

	fmt.Println("\n=== 示例完成 ===")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
