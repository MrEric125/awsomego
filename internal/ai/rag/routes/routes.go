package routes

import (
	ragh "awesome/internal/ai/rag/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterRAGRoutes 注册 RAG 相关路由
func RegisterRAGRoutes(r *gin.RouterGroup, handler *ragh.RAGHandler) {
	handler.RegisterRoutes(r)
}
