package handlers

import (
	"awesome/internal/interfaces"
	"awesome/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService interfaces.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService interfaces.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	{
		users.GET("/:id", h.GetUser)
		users.POST("/create-with-order", h.CreateUserWithOrder)
		users.POST("/complex-order", h.CreateComplexOrder)
		users.POST("/transfer", h.TransferUsers)
		users.POST("/batch", h.BatchCreateUsers)
	}
}

// GetUser 获取用户信息
// @Summary 获取用户信息
// @Description 根据用户ID获取用户信息
// @Tags 用户
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	name := h.userService.SGetUserName(id)
	c.JSON(http.StatusOK, gin.H{
		"user_id":   id,
		"user_name": name,
	})
}

// CreateUserWithOrder 创建用户并创建订单（使用事务）
// @Summary 创建用户和订单
// @Description 在一个事务中创建用户和订单
// @Tags 用户
// @Param request body CreateUserWithOrderRequest true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/users/create-with-order [post]
func (h *UserHandler) CreateUserWithOrder(c *gin.Context) {
	var req CreateUserWithOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	err := h.userService.CreateUserWithOrder(
		req.Name,
		req.Email,
		req.Age,
		req.ProductName,
		req.Amount,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user with order: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user and order created successfully",
	})
}

// CreateComplexOrder 创建复杂订单（使用事务，涉及多表操作）
// @Summary 创建复杂订单
// @Description 在一个事务中检查用户、检查库存、减少库存、创建订单
// @Tags 订单
// @Param request body CreateComplexOrderRequest true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/users/complex-order [post]
func (h *UserHandler) CreateComplexOrder(c *gin.Context) {
	var req CreateComplexOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	err := h.userService.CreateComplexOrder(
		req.UserID,
		req.ProductID,
		req.Quantity,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create complex order: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "complex order created successfully",
	})
}

// TransferUsers 用户间转账（使用事务）
// @Summary 用户间转账
// @Description 在一个事务中执行转账操作
// @Tags 用户
// @Param request body TransferUsersRequest true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/users/transfer [post]
func (h *UserHandler) TransferUsers(c *gin.Context) {
	var req TransferUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	err := h.userService.TransferUsers(
		req.FromUserID,
		req.ToUserID,
		req.Amount,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to transfer: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "transfer completed successfully",
	})
}

// BatchCreateUsers 批量创建用户（使用事务）
// @Summary 批量创建用户
// @Description 在一个事务中批量创建用户
// @Tags 用户
// @Param request body BatchCreateUsersRequest true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/users/batch [post]
func (h *UserHandler) BatchCreateUsers(c *gin.Context) {
	var req BatchCreateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	err := h.userService.BatchCreateUsers(req.Users)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to batch create users: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "users created successfully",
		"count":   len(req.Users),
	})
}

// 请求结构体定义
type CreateUserWithOrderRequest struct {
	Name        string  `json:"name" binding:"required"`
	Email       string  `json:"email" binding:"required,email"`
	Age         int     `json:"age" binding:"required"`
	ProductName string  `json:"product_name" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
}

type CreateComplexOrderRequest struct {
	UserID    int `json:"user_id" binding:"required"`
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required"`
}

type TransferUsersRequest struct {
	FromUserID int     `json:"from_user_id" binding:"required"`
	ToUserID   int     `json:"to_user_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
}

type BatchCreateUsersRequest struct {
	Users []models.User `json:"users" binding:"required"`
}
