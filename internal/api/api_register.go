package api

import (
	"awesome/internal/handlers"
	"awesome/internaldig"
	"fmt"
	"github.com/gin-gonic/gin"

	"log"
)

func Init(r *gin.Engine) {

	// 获取 AIHandler
	aiHandler, err := internaldig.Get[*handlers.ChatHandler]()
	if err != nil {
		log.Printf("Warning: failed to get AIHandler: %v", err)
	}
	//
	//// 获取 LangChainHandler
	//lcHandler, err := internaldig.Get[*handlers.LangChainHandler]()
	//if err != nil {
	//	log.Printf("Warning: failed to get LangChainHandler: %v", err)
	//}

	//// 获取 RAGHandler
	//ragHandler, err := internaldig.Get[*handlers.RAGHandler]()
	//if err != nil {
	//	log.Printf("Warning: failed to get RAGHandler: %v", err)
	//}

	// 注册基础路由
	r.GET("/ping", handlers.Ping)
	r.GET("/health", handlers.HealthCheck)

	// 注册 AI 相关路由（Eino）
	if aiHandler != nil {
		aiHandler.RegisterRoutes(r.Group("/api"))
		fmt.Println("AI routes (Eino) registered successfully")
	}
	//
	//// 注册 LangChain 相关路由
	//if lcHandler != nil {
	//	lcGroup := r.Group("/api/lc")
	//	lcGroup.POST("/chat", lcHandler.Chat)
	//	lcGroup.POST("/chain", lcHandler.RunChain)
	//	lcGroup.POST("/summarize", lcHandler.Summarize)
	//	lcGroup.POST("/translate", lcHandler.Translate)
	//	lcGroup.POST("/qa", lcHandler.QuestionAnswering)
	//	fmt.Println("LangChain routes registered successfully")
	//}
	//
	//// 注册 RAG 相关路由
	//if ragHandler != nil {
	//	ragroutes.RegisterRAGRoutes(r.Group("/api"), ragHandler)
	//	fmt.Println("RAG routes registered successfully")
	//}

}
