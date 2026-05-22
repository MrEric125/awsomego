package handlers

import (
	"net/http"

	"awesome/internal/ai/rag/service"

	"github.com/gin-gonic/gin"
)

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragService service.RAGService
}

// NewRAGHandler 创建 RAG 处理器
func NewRAGHandler(ragService service.RAGService) *RAGHandler {
	return &RAGHandler{
		ragService: ragService,
	}
}

// RegisterRoutes 注册 RAG 路由
func (h *RAGHandler) RegisterRoutes(r *gin.RouterGroup) {
	rag := r.Group("/rag")
	{
		rag.POST("/documents", h.AddDocuments)
		rag.POST("/text", h.AddText)
		rag.POST("/query", h.Query)
		rag.POST("/query/history", h.QueryWithHistory)
		rag.POST("/query/stream", h.StreamQuery)
		rag.DELETE("/clear", h.Clear)
	}
}

// AddDocumentsRequest 添加文档请求
type AddDocumentsRequest struct {
	Documents []struct {
		Content  string         `json:"content" binding:"required"`
		Metadata map[string]any `json:"metadata,omitempty"`
	} `json:"documents" binding:"required"`
}

// AddDocuments 添加文档到知识库
// @Summary 添加文档到知识库
// @Description 将文档添加到 RAG 知识库中
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body AddDocumentsRequest true "文档列表"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /rag/documents [post]
func (h *RAGHandler) AddDocuments(c *gin.Context) {
	var req AddDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为服务层文档格式
	docs := make([]*service.Document, len(req.Documents))
	for i, doc := range req.Documents {
		docs[i] = &service.Document{
			Content:  doc.Content,
			Metadata: doc.Metadata,
		}
	}

	// 添加文档
	if err := h.ragService.AddDocuments(c.Request.Context(), docs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Documents added successfully",
		"count":   len(docs),
	})
}

// AddTextRequest 添加文本请求
type AddTextRequest struct {
	Text     string         `json:"text" binding:"required"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AddText 添加文本到知识库
// @Summary 添加文本到知识库
// @Description 将文本添加到 RAG 知识库中
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body AddTextRequest true "文本内容"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /rag/text [post]
func (h *RAGHandler) AddText(c *gin.Context) {
	var req AddTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 添加文本
	if err := h.ragService.AddText(c.Request.Context(), req.Text, req.Metadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Text added successfully",
	})
}

// QueryRequest 查询请求
type QueryRequest struct {
	Question string `json:"question" binding:"required"`
}

// Query RAG 查询
// @Summary RAG 查询
// @Description 基于知识库进行 RAG 查询
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body QueryRequest true "问题"
// @Success 200 {object} service.RAGResponse
// @Failure 400 {object} map[string]interface{}
// @Router /rag/query [post]
func (h *RAGHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 执行查询
	response, err := h.ragService.Query(c.Request.Context(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryWithHistoryRequest 带历史记录的查询请求
type QueryWithHistoryRequest struct {
	Question string            `json:"question" binding:"required"`
	History  []service.Message `json:"history,omitempty"`
}

// QueryWithHistory 带历史记录的 RAG 查询
// @Summary 带历史记录的 RAG 查询
// @Description 基于知识库和历史记录进行 RAG 查询
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body QueryWithHistoryRequest true "问题和历史记录"
// @Success 200 {object} service.RAGResponse
// @Failure 400 {object} map[string]interface{}
// @Router /rag/query/history [post]
func (h *RAGHandler) QueryWithHistory(c *gin.Context) {
	var req QueryWithHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 执行查询
	response, err := h.ragService.QueryWithHistory(c.Request.Context(), req.Question, req.History)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// StreamQuery 流式 RAG 查询
// @Summary 流式 RAG 查询
// @Description 基于知识库进行流式 RAG 查询
// @Tags RAG
// @Accept json
// @Produce text/event-stream
// @Param request body QueryRequest true "问题"
// @Success 200 {string} string "SSE stream"
// @Failure 400 {object} map[string]interface{}
// @Router /rag/query/stream [post]
func (h *RAGHandler) StreamQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 执行流式查询
	stream, err := h.ragService.StreamQuery(c.Request.Context(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 发送流式响应
	for chunk := range stream {
		c.SSEvent("message", chunk)
		c.Writer.Flush()
	}
}

// Clear 清空知识库
// @Summary 清空知识库
// @Description 清空 RAG 知识库中的所有文档
// @Tags RAG
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /rag/clear [delete]
func (h *RAGHandler) Clear(c *gin.Context) {
	if err := h.ragService.Clear(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Knowledge base cleared successfully",
	})
}
