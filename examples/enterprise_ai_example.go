package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"awesome/internal/ai/config"
	"awesome/internal/ai/openai"
	"awesome/internal/ai/service"
)

func main() {
	// 加载配置
	cfg := config.LoadAIConfig()

	// 示例 1: 直接使用 OpenAI 客户端
	fmt.Println("=== 示例 1: 直接使用 OpenAI 客户端 ===")
	directClientExample(cfg)

	// 示例 2: 使用企业级服务
	fmt.Println("\n=== 示例 2: 使用企业级服务 ===")
	enterpriseServiceExample(cfg)

	// 示例 3: 流式聊天
	fmt.Println("\n=== 示例 3: 流式聊天 ===")
	streamChatExample(cfg)

	// 示例 4: 创建嵌入
	fmt.Println("\n=== 示例 4: 创建嵌入 ===")
	embeddingExample(cfg)
}

// directClientExample 直接使用 OpenAI 客户端示例
func directClientExample(cfg *config.AIConfig) {
	// 创建客户端
	client, err := openai.NewClient(cfg)
	if err != nil {
		log.Printf("创建客户端失败: %v", err)
		return
	}
	defer client.Close()

	// 构建请求
	req := &openai.ChatCompletionRequest{
		Model: "qwen3_32b",
		Messages: []openai.ChatMessage{
			{Role: "system", Content: "你是一个有用的AI助手。"},
			{Role: "user", Content: "请用一句话介绍Go语言。"},
		},
	}

	// 调用 API
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}

	fmt.Printf("响应: %s\n", resp.Choices[0].Message.Content)
	fmt.Printf("使用令牌: 输入=%d, 输出=%d, 总计=%d\n",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens)

	// 获取统计信息
	stats := client.GetStats()
	fmt.Printf("客户端统计: 总请求=%d, 成功=%d, 失败=%d\n",
		stats.TotalRequests, stats.SuccessCount, stats.FailureCount)
}

// enterpriseServiceExample 企业级服务示例
func enterpriseServiceExample(cfg *config.AIConfig) {
	// 创建企业级服务
	svc, err := service.NewEnterpriseAIService(cfg,
		service.WithMaxConcurrency(50),
		service.WithRateLimit(100, 20),
		service.WithCache(true, 1000, 5*time.Minute),
	)
	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}
	defer svc.Close()

	// 构建请求
	req := &service.ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatMessage{
			{Role: "system", Content: "你是一个专业的编程助手。"},
			{Role: "user", Content: "什么是微服务架构？"},
		},
		UseCache: true,
	}

	// 调用服务
	ctx := context.Background()
	resp, err := svc.Chat(ctx, req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}

	fmt.Printf("响应: %s\n", resp.Content)
	fmt.Printf("延迟: %dms, 缓存: %v\n", resp.LatencyMs, resp.Cached)

	// 获取服务统计
	stats := svc.GetStats()
	fmt.Printf("服务统计: 总请求=%d, 成功=%d, 失败=%d\n",
		stats.TotalRequests, stats.SuccessCount, stats.FailureCount)
}

// streamChatExample 流式聊天示例
func streamChatExample(cfg *config.AIConfig) {
	// 创建服务
	svc, err := service.NewEnterpriseAIService(cfg)
	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}
	defer svc.Close()

	// 构建请求
	req := &service.ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatMessage{
			{Role: "user", Content: "请写一首关于编程的短诗。"},
		},
	}

	// 流式调用
	ctx := context.Background()
	stream, err := svc.StreamChat(ctx, req)
	if err != nil {
		log.Printf("流式请求失败: %v", err)
		return
	}

	fmt.Print("流式响应: ")
	for chunk := range stream {
		if chunk.Error != nil {
			log.Printf("流错误: %v", chunk.Error)
			break
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()
}

// embeddingExample 嵌入示例
func embeddingExample(cfg *config.AIConfig) {
	// 创建服务
	svc, err := service.NewEnterpriseAIService(cfg)
	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}
	defer svc.Close()

	// 构建请求
	req := &service.EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: []string{
			"Go语言是一门静态类型的编程语言",
			"Python是一门动态类型的编程语言",
		},
	}

	// 调用服务
	ctx := context.Background()
	resp, err := svc.CreateEmbedding(ctx, req)
	if err != nil {
		log.Printf("嵌入请求失败: %v", err)
		return
	}

	fmt.Printf("创建了 %d 个嵌入向量\n", len(resp.Data))
	for i, data := range resp.Data {
		fmt.Printf("向量 %d: 维度=%d, 前5个值=%v...\n",
			i, len(data.Embedding), data.Embedding[:5])
	}
}

//
//// 环境变量设置示例
//func printEnvExample() {
//	fmt.Println("=== 环境变量配置示例 ===")
//	fmt.Println("请设置以下环境变量:")
//	fmt.Println("  export OPENAI_API_KEY=your-api-key")
//	fmt.Println("  export OPENAI_MODEL_NAME=gpt-4")
//	fmt.Println("  export OPENAI_BASE_URL=https://api.openai.com/v1")
//	fmt.Println("  export AI_TIMEOUT=30")
//	fmt.Println("  export AI_MAX_RETRIES=3")
//	fmt.Println("  export AI_TEMPERATURE=0.7")
//	fmt.Println("  export AI_MAX_TOKENS=2000")
//}
//
//// init 初始化检查
//func init() {
//	// 检查 API Key
//	if os.Getenv("OPENAI_API_KEY") == "" {
//		fmt.Println("警告: OPENAI_API_KEY 环境变量未设置")
//		fmt.Println("请设置环境变量或使用示例配置")
//		printEnvExample()
//		os.Exit(1)
//	}
//}
