package main

import (
	"awesome/internal"
	_ "awesome/internal"
	aih "awesome/internal/ai/handlers"
	ragh "awesome/internal/ai/rag/handlers"
	ragroutes "awesome/internal/ai/rag/routes"
	"awesome/internal/handlers"
	lch "awesome/internal/langchain/handlers"
	"awesome/internaldig"
	"fmt"
	"github.com/joho/godotenv"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化 AI 模块（Eino + RAG）

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	internal.Init()

	// 获取 UserHandler
	userHandler, err := internaldig.Get[*handlers.UserHandler]()
	if err != nil {
		log.Fatalf("Failed to get UserHandler: %v", err)
	}

	// 获取 AIHandler
	aiHandler, err := internaldig.Get[*aih.AIHandler]()
	if err != nil {
		log.Printf("Warning: failed to get AIHandler: %v", err)
	}

	// 获取 LangChainHandler
	lcHandler, err := internaldig.Get[*lch.LangChainHandler]()
	if err != nil {
		log.Printf("Warning: failed to get LangChainHandler: %v", err)
	}

	// 获取 RAGHandler
	ragHandler, err := internaldig.Get[*ragh.RAGHandler]()
	if err != nil {
		log.Printf("Warning: failed to get RAGHandler: %v", err)
	}

	// 创建 Gin 引擎
	r := gin.Default()

	// 注册基础路由
	r.GET("/ping", handlers.Ping)
	r.GET("/health", handlers.HealthCheck)

	// 注册用户相关路由
	userHandler.RegisterRoutes(r)

	// 注册 AI 相关路由（Eino）
	if aiHandler != nil {
		aiHandler.RegisterRoutes(r.Group("/api"))
		fmt.Println("AI routes (Eino) registered successfully")
	}

	// 注册 LangChain 相关路由
	if lcHandler != nil {
		lcGroup := r.Group("/api/lc")
		lcGroup.POST("/chat", lcHandler.Chat)
		lcGroup.POST("/chain", lcHandler.RunChain)
		lcGroup.POST("/summarize", lcHandler.Summarize)
		lcGroup.POST("/translate", lcHandler.Translate)
		lcGroup.POST("/qa", lcHandler.QuestionAnswering)
		fmt.Println("LangChain routes registered successfully")
	}

	// 注册 RAG 相关路由
	if ragHandler != nil {
		ragroutes.RegisterRAGRoutes(r.Group("/api"), ragHandler)
		fmt.Println("RAG routes registered successfully")
	}

	// 启动服务器
	fmt.Println("Server is starting on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
