package langchain

import (
	"awesome/internal/handlers"
	"awesome/internal/inf/ai/new/langchain/components"
	lcconfig "awesome/internal/inf/ai/new/langchain/config"
	"awesome/internal/inf/ai/new/langchain/service"
	"awesome/internaldig"
	"fmt"
)

// Init 初始化 LangChain 模块并注册到依赖注入容器
func Init() {
	// 1. 加载 LangChain 配置
	cfg := lcconfig.LoadLangChainConfig()

	// 2. 注册 LLMProvider
	if err := internaldig.Provide(func() *components.LLMProvider {
		return components.NewLLMProvider(cfg)
	}); err != nil {
		fmt.Println("Error providing LLMProvider:", err)
		panic(err)
	}

	// 3. 注册 LangChainService
	if err := internaldig.Provide(func(provider *components.LLMProvider) (service.LangChainService, error) {
		return service.NewLangChainService(provider)
	}); err != nil {
		fmt.Println("Error providing LangChainService:", err)
		panic(err)
	}

	// 4. 注册 LangChainHandler
	if err := internaldig.Provide(func(lcService service.LangChainService) *handlers.LangChainHandler {
		return handlers.NewLangChainHandler(lcService)
	}); err != nil {
		fmt.Println("Error providing LangChainHandler:", err)
		panic(err)
	}

	fmt.Println("LangChain module initialized successfully")
}
