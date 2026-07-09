package main

import (
	"awesome/internal"
	_ "awesome/internal"
	"awesome/internal/api"
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

	// 创建 Gin 引擎
	r := gin.Default()

	api.Init(r)
	// 启动服务器
	fmt.Println("Server is starting on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
