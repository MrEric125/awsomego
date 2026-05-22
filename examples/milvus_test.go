package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"awesome/internal/ai/components"
	"awesome/internal/ai/config"
	ragComponents "awesome/internal/ai/rag/components"
	ragConfig "awesome/internal/ai/rag/config"
	ragService "awesome/internal/ai/rag/service"
)

func TestMilvus(t *testing.T) {
	// 1. 加载配置
	aiCfg := config.LoadAIConfig()
	ragCfg := ragConfig.LoadRAGConfig()

	// 配置使用 Milvus
	ragCfg.VectorStoreType = "milvus"
	ragCfg.MilvusHost = "localhost"
	ragCfg.MilvusPort = 19530
	ragCfg.MilvusCollection = "rag_knowledge_base"

	// 2. 创建聊天模型
	chatModelProvider := components.NewChatModelProvider(aiCfg)
	chatModel, err := chatModelProvider.GetDefaultChatModel()
	if err != nil {
		log.Fatalf("Failed to create chat model: %v", err)
	}

	// 3. 创建 Embedding 模型
	embeddingProvider := ragComponents.NewEmbeddingProvider(ragCfg)
	embeddingModel, err := embeddingProvider.GetEmbeddingModel()
	if err != nil {
		log.Printf("Warning: Failed to get embedding model, using mock: %v", err)
		embeddingModel = &ragComponents.MockEmbedding{}
		embeddingModel.SetDimension(ragCfg.VectorDimension)
	}

	// 4. 创建 Milvus 向量存储
	vectorStoreProvider := ragComponents.NewVectorStoreProvider(ragCfg, embeddingModel)
	vectorStore, err := vectorStoreProvider.GetVectorStore()
	if err != nil {
		log.Fatalf("Failed to create Milvus vector store: %v", err)
	}

	// 5. 创建 RAG 服务
	ragSvc := ragService.NewRAGService(chatModel, vectorStore, ragCfg)

	ctx := context.Background()

	// 6. 添加文档到 Milvus
	fmt.Println("=== 添加文档到 Milvus ===")
	docs := []*ragService.Document{
		{
			Content: "Milvus 是一个开源的向量数据库，专为海量向量数据的存储、索引和检索而设计。它支持多种索引类型，包括 IVF、HNSW、ANNOY 等。",
			Metadata: map[string]any{
				"source": "milvus_intro.txt",
				"type":   "database",
			},
		},
		{
			Content: "向量数据库在 AI 应用中扮演重要角色，特别是在 RAG（检索增强生成）系统中。它们能够高效地进行相似度搜索，支持大规模向量数据的实时查询。",
			Metadata: map[string]any{
				"source": "vector_db_intro.txt",
				"type":   "ai_technology",
			},
		},
		{
			Content: "Milvus 支持分布式部署，可以轻松扩展到多个节点。它提供了丰富的 API 接口，支持 Python、Go、Java 等多种编程语言。",
			Metadata: map[string]any{
				"source": "milvus_features.txt",
				"type":   "database",
			},
		},
	}

	if err := ragSvc.AddDocuments(ctx, docs); err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Println("文档添加成功！")

	// 7. 执行 RAG 查询
	fmt.Println("\n=== 执行 RAG 查询 ===")
	questions := []string{
		"什么是 Milvus？",
		"向量数据库有什么用途？",
		"Milvus 支持哪些编程语言？",
	}

	for _, question := range questions {
		fmt.Printf("\n问题: %s\n", question)
		response, err := ragSvc.Query(ctx, question)
		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		fmt.Printf("回答: %s\n", response.Answer)
		fmt.Printf("置信度: %.2f\n", response.Confidence)
		if len(response.Sources) > 0 {
			fmt.Printf("来源文档数量: %d\n", len(response.Sources))
			for i, source := range response.Sources {
				fmt.Printf("  来源 %d: %s (分数: %.2f)\n", i+1, truncate(source.Content, 50), source.Score)
			}
		}
	}

	// 8. 流式查询示例
	fmt.Println("\n=== 流式查询示例 ===")
	stream, err := ragSvc.StreamQuery(ctx, "请详细介绍 Milvus 的主要特性")
	if err != nil {
		log.Printf("Stream query failed: %v", err)
	} else {
		fmt.Print("流式回答: ")
		for chunk := range stream {
			fmt.Print(chunk)
		}
		fmt.Println()
	}

	// 9. 带历史记录的查询
	fmt.Println("\n=== 带历史记录的查询 ===")
	history := []ragService.Message{
		{
			Role:    "user",
			Content: "什么是向量数据库？",
		},
		{
			Role:    "assistant",
			Content: "向量数据库是专门用于存储和检索向量嵌入的数据库，支持高效的相似度搜索。",
		},
	}

	response, err := ragSvc.QueryWithHistory(ctx, "它在 RAG 系统中有什么作用？", history)
	if err != nil {
		log.Printf("Query with history failed: %v", err)
	} else {
		fmt.Printf("回答: %s\n", response.Answer)
	}

	// 注意：不清空数据，保留知识库内容
	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("数据已保存在 Milvus 中，可以继续使用")
}
