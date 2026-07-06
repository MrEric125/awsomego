package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"awesome/internal/ai/components"
	aiconfig "awesome/internal/ai/config"
	"awesome/internal/ai/service"
)

// 这个示例展示如何使用 Eino 框架进行 AI 应用开发
func TestEinoExample(t *testing.T) {
	fmt.Println("=== Eino AI Framework Examples ===")

	// 1. 加载配置
	cfg := aiconfig.LoadAIConfig()

	// 检查是否配置了 API Key
	if cfg.ArkAPIKey == "" {
		log.Println("Warning: ARK_API_KEY is not set. Please set environment variable.")
		log.Println("Example: export ARK_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// 2. 创建 ChatModel 提供者
	provider := components.NewChatModelProvider(cfg)

	// 3. 创建 AI 服务
	aiService, err := service.NewAIService(provider)
	if err != nil {
		log.Fatalf("Failed to create AI service: %v", err)
	}

	ctx := context.Background()

	// 示例 1: 简单对话
	fmt.Println("📝 Example 1: Simple Chat")
	fmt.Println("-----------------------------------")
	reply, err := aiService.Chat(ctx, "Go 语言的优势是什么？")
	if err != nil {
		log.Printf("Chat error: %v\n", err)
	} else {
		fmt.Printf("Q: Go 语言的优势是什么？\n")
		fmt.Printf("A: %s\n\n", reply)
	}

	// 示例 2: 带历史记录的对话
	fmt.Println("💬 Example 2: Chat with History")
	fmt.Println("-----------------------------------")
	history := []service.Message{
		{Role: "system", Content: "你是一个专业的编程助手。"},
		{Role: "user", Content: "什么是微服务架构？"},
		{Role: "assistant", Content: "微服务架构是一种将应用程序构建为一组小型服务的方法，每个服务运行在独立的进程中..."},
		{Role: "user", Content: "它有什么优点？"},
	}

	reply, err = aiService.ChatWithHistory(ctx, history)
	if err != nil {
		log.Printf("Chat with history error: %v\n", err)
	} else {
		fmt.Printf("Q: 它有什么优点？\n")
		fmt.Printf("A: %s\n\n", reply)
	}

	// 示例 3: 文本摘要
	fmt.Println("📄 Example 3: Text Summarization")
	fmt.Println("-----------------------------------")
	longText := `Go（又称Golang）是Google开发的一种静态强类型、编译型、并发型，并具有垃圾回收功能的编程语言。
罗伯特·格瑞史莫、罗勃·派克及肯·汤普逊于2007年9月开始设计Go，稍后Ian Lance Taylor、Russ Cox加入项目。
Go是基于Inferno操作系统所开发的。Go于2009年11月正式宣布推出，成为开放源代码项目，并在Linux及Mac OS X平台上进行了实现，后来扩充到Windows平台。
Go的语法接近C语言，但对于变量的声明有所不同。Go支持垃圾回收功能。Go的并行模型是以东尼·霍尔的通信顺序进程（CSP）为基础，
这种类似模型的其他实现包括Erlang等。但是Go的可组合性模型则是基于类型系统而非传统的类继承。`

	summary, err := aiService.Summarize(ctx, longText)
	if err != nil {
		log.Printf("Summarize error: %v\n", err)
	} else {
		fmt.Printf("Original text length: %d characters\n", len(longText))
		fmt.Printf("Summary: %s\n\n", summary)
	}

	// 示例 4: 翻译
	fmt.Println("🌍 Example 4: Translation")
	fmt.Println("-----------------------------------")
	text := "Hello, World! Welcome to the future of AI development with Go and Eino."
	translation, err := aiService.Translate(ctx, text, "中文")
	if err != nil {
		log.Printf("Translate error: %v\n", err)
	} else {
		fmt.Printf("Original: %s\n", text)
		fmt.Printf("Translation: %s\n\n", translation)
	}

	// 示例 5: 流式对话
	fmt.Println("🌊 Example 5: Streaming Chat")
	fmt.Println("-----------------------------------")
	fmt.Println("Q: 请介绍一下云计算的三个主要服务模式")
	fmt.Print("A: ")

	stream, err := aiService.StreamChat(ctx, "请介绍一下云计算的三个主要服务模式")
	if err != nil {
		log.Printf("Stream chat error: %v\n", err)
	} else {
		for chunk := range stream {
			fmt.Print(chunk)
		}
		fmt.Println("==========")
	}

	fmt.Println("=== Examples Completed ===")
}
