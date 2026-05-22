package repository

import (
	"context"
	"fmt"

	"awesome/internal/interfaces"
	"awesome/internal/models"

	"gorm.io/gorm"
)

// OrderRepositoryImpl 订单仓库实现
type OrderRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewOrderRepository 创建订单仓库实例
func NewOrderRepository(db *gorm.DB) interfaces.OrderRepository {
	return &OrderRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db),
	}
}

// Create 创建订单
func (r *OrderRepositoryImpl) Create(ctx context.Context, order *models.Order) error {
	return r.BaseRepositoryImpl.Create(ctx, order)
}

// GetByID 根据ID获取订单
func (r *OrderRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Order, error) {
	var order models.Order
	db := r.getDB(ctx)
	if err := db.Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &order, nil
}

// GetByUserID 根据用户ID获取订单列表
func (r *OrderRepositoryImpl) GetByUserID(ctx context.Context, userID int, page, pageSize int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	return orders, total, nil
}

// UpdateStatus 更新订单状态
func (r *OrderRepositoryImpl) UpdateStatus(ctx context.Context, id int, status int) error {
	db := r.getDB(ctx)
	result := db.Model(&models.Order{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update order status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("order not found: %d", id)
	}

	return nil
}

// Delete 删除订单（软删除）
func (r *OrderRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.BaseRepositoryImpl.Delete(ctx, &models.Order{}, id)
}

// BatchCreate 批量创建订单
func (r *OrderRepositoryImpl) BatchCreate(ctx context.Context, orders []*models.Order) error {
	db := r.getDB(ctx)
	if err := db.CreateInBatches(orders, 100).Error; err != nil {
		return fmt.Errorf("failed to batch create orders: %w", err)
	}
	return nil
}
