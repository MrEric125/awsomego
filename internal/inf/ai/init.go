package ai

import (
	"awesome/internal/inf/ai/new/config"
	"awesome/internal/inf/ai/new/service"
	"awesome/internal/inf/ai/rag"
	"awesome/internaldig"
	"fmt"
)

// Init 初始化 AI 模块并注册到依赖注入容器
func Init() {
	// 1. 加载 AI 配置
	//cfg := aiconfig.LoadAIConfig()

	// 4. 注册 AIService
	if err := internaldig.Provide(func() (*service.ChatService, error) {

		return service.NewChatService(config.Config{})
	}); err != nil {
		fmt.Println("Error providing AIService:", err)
		panic(err)
	}

	// 6. 初始化 RAG 模块
	if err := rag.RegisterRAGServices(internaldig.DigContainer); err != nil {
		fmt.Println("Error initializing RAG module:", err)
		panic(err)
	}

	fmt.Println("AI module initialized successfully (with RAG support)")
}
