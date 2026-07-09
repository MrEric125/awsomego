package ai

import (
	"awesome/internal/handlers"
	"awesome/internal/inf/ai/components"
	aiconfig "awesome/internal/inf/ai/config"
	"awesome/internal/inf/ai/rag"
	"awesome/internal/inf/ai/service"
	"awesome/internaldig"
	"fmt"

	"github.com/cloudwego/eino/components/model"
)

// Init 初始化 AI 模块并注册到依赖注入容器
func Init() {
	// 1. 加载 AI 配置
	cfg := aiconfig.LoadAIConfig()

	// 2. 注册 ChatModelProvider
	if err := internaldig.Provide(func() *components.ChatModelProvider {
		return components.NewChatModelProvider(cfg)
	}); err != nil {
		fmt.Println("Error providing ChatModelProvider:", err)
		panic(err)
	}

	// 3. 注册 ChatModel
	if err := internaldig.Provide(func(provider *components.ChatModelProvider) (model.ChatModel, error) {
		return provider.GetDefaultChatModel()
	}); err != nil {
		fmt.Println("Error providing ChatModel:", err)
		panic(err)
	}

	// 4. 注册 AIService
	if err := internaldig.Provide(func(provider *components.ChatModelProvider) (service.AIService, error) {
		return service.NewAIService(provider)
	}); err != nil {
		fmt.Println("Error providing AIService:", err)
		panic(err)
	}

	// 5. 注册 AIHandler
	if err := internaldig.Provide(func(aiService service.AIService) *handlers.AIHandler {
		return handlers.NewAIHandler(aiService)
	}); err != nil {
		fmt.Println("Error providing AIHandler:", err)
		panic(err)
	}

	// 6. 初始化 RAG 模块
	if err := rag.RegisterRAGServices(internaldig.DigContainer); err != nil {
		fmt.Println("Error initializing RAG module:", err)
		panic(err)
	}

	fmt.Println("AI module initialized successfully (with RAG support)")
}
