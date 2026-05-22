package repository

import (
	"context"
	"fmt"

	"awesome/internal/models"

	"gorm.io/gorm"
)

// ProductRepositoryImpl 产品仓库实现
type ProductRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewProductRepository 创建产品仓库实例
func NewProductRepository(db *gorm.DB) *ProductRepositoryImpl {
	return &ProductRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db),
	}
}

// Create 创建产品
func (r *ProductRepositoryImpl) Create(ctx context.Context, product *models.Product) error {
	return r.BaseRepositoryImpl.Create(ctx, product)
}

// GetByID 根据ID获取产品
func (r *ProductRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Product, error) {
	var product models.Product
	if err := r.FindByID(ctx, &product, id); err != nil {
		return nil, err
	}
	return &product, nil
}

// UpdateStock 更新产品库存
func (r *ProductRepositoryImpl) UpdateStock(ctx context.Context, id int, stock int) error {
	db := r.getDB(ctx)
	result := db.Model(&models.Product{}).Where("id = ?", id).Update("stock", stock)
	if result.Error != nil {
		return fmt.Errorf("failed to update product stock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("product not found: %d", id)
	}

	return nil
}

// DecreaseStock 减少库存（用于下单场景，原子操作）
func (r *ProductRepositoryImpl) DecreaseStock(ctx context.Context, productID int, quantity int) error {
	db := r.getDB(ctx)
	// 使用原子操作减少库存，确保库存不会为负
	result := db.Model(&models.Product{}).
		Where("id = ? AND stock >= ?", productID, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))

	if result.Error != nil {
		return fmt.Errorf("failed to decrease stock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("product not found or insufficient stock: %d", productID)
	}

	return nil
}
