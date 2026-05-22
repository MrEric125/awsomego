package internal

import (
	"awesome/internal/ai"
	"awesome/internal/database"
	"awesome/internal/handlers"
	"awesome/internal/langchain"
	"awesome/internal/repository"
	"awesome/internal/service"
)

func Init() {
	// 初始化数据库连接并执行迁移
	database.InitDatabase()

	// 可选：创建示例数据（开发环境）
	// database.CreateSampleData(db)

	repository.Init()
	service.Init()
	handlers.Init()

	// 初始化 AI 模块
	ai.Init()
	// 初始化 LangChain 模块
	langchain.Init()
}
