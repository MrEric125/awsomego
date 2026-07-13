package init

import (
	"awesome/internal/database"
	"awesome/internal/handlers"
	"awesome/internal/inf/ai/new/config"
	"awesome/internal/inf/ai/new/service"
	"awesome/internaldig"
	"fmt"
)

func init() {
	InitDb()
	InitService()
	InitHandler()
}

func InitHandler() {
	handlers.Init()
}

func InitDb() {
	// 初始化数据库连接并执行迁移

	database.InitDatabase()

}

// Init 初始化 AI 模块并注册到依赖注入容器
func InitService() {
	// 1. 加载 AI 配置
	//cfg := aiconfig.LoadAIConfig()

	// 4. 注册 AIService
	if err := internaldig.Provide(func() (*service.ChatService, error) {

		return service.NewChatService(*config.LoadConfig())
	}); err != nil {
		fmt.Println("Error providing AIService:", err)
		panic(err)
	}

	// 6. 初始化 RAG 模块
	//if err := rag.RegisterRAGServices(internaldig.DigContainer); err != nil {
	//	fmt.Println("Error initializing RAG module:", err)
	//	panic(err)
	//}

	fmt.Println("AI module initialized successfully (with RAG support)")
	//langchain.Init()
}
