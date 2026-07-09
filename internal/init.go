package internal

import (
	"awesome/internal/database"
	"awesome/internal/inf/ai"
	"awesome/internal/inf/langchain"
)

func Init() {
	// 初始化数据库连接并执行迁移

	database.InitDatabase()

	// 初始化 AI 模块
	ai.Init()
	// 初始化 LangChain 模块
	langchain.Init()

}
