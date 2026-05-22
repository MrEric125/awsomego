package repository

import (
	"context"
	"fmt"

	"awesome/internal/interfaces"
	"awesome/internal/models"

	"gorm.io/gorm"
)

// UserRepositoryImpl 用户仓库实现
type UserRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) interfaces.UserRepository {
	return &UserRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db),
	}
}

// GetByID 根据ID获取用户
func (r *UserRepositoryImpl) GetByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User
	if err := r.FindByID(ctx, &user, id); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateName 更新用户名称
func (r *UserRepositoryImpl) UpdateName(ctx context.Context, id int, name string) error {
	db := r.getDB(ctx)
	result := db.Model(&models.User{}).Where("id = ?", id).Update("name", name)
	if result.Error != nil {
		return fmt.Errorf("failed to update user name: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	return nil
}

// Create 创建用户
func (r *UserRepositoryImpl) Create(ctx context.Context, user *models.User) error {
	return r.BaseRepositoryImpl.Create(ctx, user)
}

// Delete 删除用户（软删除）
func (r *UserRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.BaseRepositoryImpl.Delete(ctx, &models.User{}, id)
}

// ListUsers 获取用户列表
func (r *UserRepositoryImpl) ListUsers(ctx context.Context, page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	offset := (page - 1) * pageSize

	// 计数
	total, err := r.Count(ctx, &models.User{}, "1 = 1")
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	db := r.getDB(ctx)
	if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}
