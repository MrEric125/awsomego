package handlers

import (
	"awesome/internal/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查接口
func HealthCheck(c *gin.Context) {
	err := database.HealthCheck()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// Ping 简单的 ping 接口
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
