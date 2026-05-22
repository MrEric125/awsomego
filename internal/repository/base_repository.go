package repository

import (
	"context"
	"fmt"

	"awesome/internal/database"

	"gorm.io/gorm"
)

// BaseRepository 基础仓库，提供通用的数据库操作方法
type BaseRepositoryImpl struct {
	db *gorm.DB
}

// NewBaseRepository 创建基础仓库实例
func NewBaseRepository(db *gorm.DB) *BaseRepositoryImpl {
	return &BaseRepositoryImpl{db: db}
}

// getDB 获取数据库连接（优先使用事务）
func (r *BaseRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	// 尝试从 context 中获取事务
	if tx, ok := database.GetTxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	// 否则使用普通连接
	return r.db.WithContext(ctx)
}

// Create 创建记录
func (r *BaseRepositoryImpl) Create(ctx context.Context, model interface{}) error {
	db := r.getDB(ctx)
	if err := db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	return nil
}

// Update 更新记录
func (r *BaseRepositoryImpl) Update(ctx context.Context, model interface{}) error {
	db := r.getDB(ctx)
	if err := db.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}
	return nil
}

// Delete 删除记录
func (r *BaseRepositoryImpl) Delete(ctx context.Context, model interface{}, id interface{}) error {
	db := r.getDB(ctx)
	result := db.Delete(model, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}
	return nil
}

// FindByID 根据 ID 查询
func (r *BaseRepositoryImpl) FindByID(ctx context.Context, model interface{}, id interface{}) error {
	db := r.getDB(ctx)
	if err := db.First(model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("record not found: %v", id)
		}
		return fmt.Errorf("failed to find record: %w", err)
	}
	return nil
}

// Find 查询列表
func (r *BaseRepositoryImpl) Find(ctx context.Context, model interface{}, query interface{}, args ...interface{}) error {
	db := r.getDB(ctx)
	if err := db.Where(query, args...).Find(model).Error; err != nil {
		return fmt.Errorf("failed to find records: %w", err)
	}
	return nil
}

// Count 计数
func (r *BaseRepositoryImpl) Count(ctx context.Context, model interface{}, query interface{}, args ...interface{}) (int64, error) {
	db := r.getDB(ctx)
	var count int64
	if err := db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}
	return count, nil
}

// Exec 执行原生 SQL
func (r *BaseRepositoryImpl) Exec(ctx context.Context, sql string, values ...interface{}) error {
	db := r.getDB(ctx)
	if err := db.Exec(sql, values...).Error; err != nil {
		return fmt.Errorf("failed to execute sql: %w", err)
	}
	return nil
}

// Raw 执行原生查询
func (r *BaseRepositoryImpl) Raw(ctx context.Context, sql string, values ...interface{}) (*gorm.DB, error) {
	db := r.getDB(ctx)
	return db.Raw(sql, values...), nil
}
