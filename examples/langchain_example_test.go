package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	_ "awesome/internal"
	"awesome/internal/langchain/components"
	lcconfig "awesome/internal/langchain/config"
	"awesome/internal/langchain/service"
)

// LangChainExample 展示 LangChain 的使用示例
func LangChainExample() {
	fmt.Println("=== LangChain Examples ===\n")

	// 初始化配置
	cfg := lcconfig.LoadLangChainConfig()

	// 创建 LLM Provider
	provider := components.NewLLMProvider(cfg)

	// 创建 LangChain Service
	lcService, err := service.NewLangChainService(provider)
	if err != nil {
		log.Fatalf("Failed to create LangChain service: %v\n", err)
	}

	ctx := context.Background()

	// 示例 1: 简单对话
	fmt.Println("💬 Example 1: Simple Chat")
	fmt.Println("-----------------------------------")
	reply, err := lcService.Chat(ctx, "Go 语言的优势是什么？")
	if err != nil {
		log.Printf("Chat error: %v\n", err)
	} else {
		fmt.Printf("Q: Go 语言的优势是什么？\n")
		fmt.Printf("A: %s\n\n", reply)
	}

	// 示例 2: 带历史记录的对话
	fmt.Println("📚 Example 2: Chat with History")
	fmt.Println("-----------------------------------")
	messages := []service.Message{
		{Role: "user", Content: "什么是微服务架构？"},
		{Role: "assistant", Content: "微服务架构是一种将应用程序构建为一组小型服务的方法..."},
		{Role: "user", Content: "它有什么优点？"},
	}

	reply, err = lcService.ChatWithHistory(ctx, messages)
	if err != nil {
		log.Printf("Chat with history error: %v\n", err)
	} else {
		fmt.Printf("Q: 它有什么优点？\n")
		fmt.Printf("A: %s\n\n", reply)
	}

	// 示例 3: 运行不同类型的链
	fmt.Println("⛓️ Example 3: Run Different Chains")
	fmt.Println("-----------------------------------")

	// 简单链
	result, err := lcService.RunChain(ctx, "列出三个著名的编程语言", service.SimpleChain)
	if err != nil {
		log.Printf("Simple chain error: %v\n", err)
	} else {
		fmt.Printf("Simple Chain Result:\n%s\n\n", result)
	}

	// 示例 4: 文本摘要
	fmt.Println("📄 Example 4: Text Summarization")
	fmt.Println("-----------------------------------")
	longText := `Go（又称Golang）是Google开发的一种静态强类型、编译型、并发型，并具有垃圾回收功能的编程语言。
罗伯特·格瑞史莫、罗勃·派克及肯·汤普逊于2007年9月开始设计Go，稍后Ian Lance Taylor、Russ Cox加入项目。
Go是基于Inferno操作系统所开发的。Go于2009年11月正式宣布推出，成为开放源代码项目，并在Linux及Mac OS X平台上进行了实现，后来扩充到Windows平台。
Go的语法接近C语言，但对于变量的声明有所不同。Go支持垃圾回收功能。Go的并行模型是以东尼·霍尔的通信顺序进程（CSP）为基础，
这种类似模型的其他实现包括Erlang等。但是Go的可组合性模型则是基于类型系统而非传统的类继承。`

	summary, err := lcService.Summarize(ctx, longText)
	if err != nil {
		log.Printf("Summarize error: %v\n", err)
	} else {
		fmt.Printf("Original text length: %d characters\n", len(longText))
		fmt.Printf("Summary: %s\n\n", summary)
	}

	// 示例 5: 翻译
	fmt.Println("🌍 Example 5: Translation")
	fmt.Println("-----------------------------------")
	text := "Hello, World! Welcome to the future of AI development with Go and LangChain."
	translation, err := lcService.Translate(ctx, text, "中文")
	if err != nil {
		log.Printf("Translate error: %v\n", err)
	} else {
		fmt.Printf("Original: %s\n", text)
		fmt.Printf("Translation: %s\n\n", translation)
	}

	// 示例 6: 问答（基于文档）
	fmt.Println("❓ Example 6: Question Answering")
	fmt.Println("-----------------------------------")
	documents := []string{
		"Go 语言是由 Google 开发的开源编程语言。",
		"Go 语言的设计目标是简洁、高效和可靠。",
		"Go 语言内置了并发支持，使用 goroutine 和 channel。",
	}

	answer, err := lcService.QuestionAnswering(ctx, "Go 语言是谁开发的？", documents)
	if err != nil {
		log.Printf("Question answering error: %v\n", err)
	} else {
		fmt.Printf("Q: Go 语言是谁开发的？\n")
		fmt.Printf("A: %s\n\n", answer)
	}

	// 示例 7: 使用不同的 LLM 提供商
	fmt.Println("🔧 Example 7: Using Different LLM Providers")
	fmt.Println("-----------------------------------")

	// 尝试使用 OpenAI
	if cfg.OpenAIAPIKey != "" {
		openaiLLM, err := provider.GetOpenAILLM()
		if err != nil {
			log.Printf("Get OpenAI LLM error: %v\n", err)
		} else {
			fmt.Printf("✓ OpenAI LLM initialized successfully (Model: %s)\n", cfg.OpenAIModelName)
			_ = openaiLLM // 使用 llm
		}
	} else {
		fmt.Println("⚠ OpenAI API Key not configured, skipping OpenAI example")
	}

	// 尝试使用 Ollama（本地模型）
	ollamaLLM, err := provider.GetOllamaLLM()
	if err != nil {
		log.Printf("Get Ollama LLM error: %v (Ollama may not be running)\n", err)
	} else {
		fmt.Printf("✓ Ollama LLM initialized successfully (Model: %s)\n", cfg.OllamaModel)
		_ = ollamaLLM // 使用 llm
	}

	fmt.Println("\n=== Examples Completed ===")
	fmt.Println("\n💡 Tips:")
	fmt.Println("- 设置 OPENAI_API_KEY 环境变量以使用 OpenAI")
	fmt.Println("- 安装并启动 Ollama 以使用本地模型")
	fmt.Println("- 查看 internal/langchain 目录了解更多实现细节")

	// 示例 8: 列出可用的提供商
	fmt.Println("\n🔍 Example 8: List Available Providers")
	fmt.Println("-----------------------------------")
	availableProviders := lcService.ListProviders()
	fmt.Printf("Available providers: %v\n", availableProviders)

	// 示例 9: 使用特定提供商对话
	if len(availableProviders) > 0 {
		fmt.Println("\n🎯 Example 9: Chat with Specific Provider")
		fmt.Println("-----------------------------------")
		provider := availableProviders[0] // 使用第一个可用的提供商
		reply, err := lcService.ChatWithProvider(ctx, "Hello from LangChain!", provider)
		if err != nil {
			log.Printf("Chat with %s error: %v\n", provider, err)
		} else {
			fmt.Printf("Using provider: %s\n", provider)
			fmt.Printf("Reply: %s\n", reply)
		}
	}
}

func TestLangchain(t *testing.T) {
	LangChainExample()
}
