package examples

import (
	"context"
	"fmt"

	"awesome/internal/database"
	"awesome/internal/interfaces"
	"awesome/internal/models"
	"awesome/internal/repository"

	"gorm.io/gorm"
)

// ExampleSimpleTransaction 简单事务示例
func ExampleSimpleTransaction() {
	db := database.GetDB()

	// 使用 GORM 内置的事务方法
	err := db.Transaction(func(tx *gorm.DB) error {
		// 在事务中执行多个操作
		user := &models.User{
			Name:  "张三",
			Email: "zhangsan@example.com",
			Age:   25,
		}
		if err := tx.Create(user).Error; err != nil {
			return err // 返回错误会触发回滚
		}

		order := &models.Order{
			UserID:  user.ID,
			Product: "iPhone 15",
			Amount:  7999.00,
			Status:  0,
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		return nil // 返回 nil 会提交事务
	})

	if err != nil {
		fmt.Printf("事务失败: %v\n", err)
		return
	}

	fmt.Println("事务成功")
}

// ExampleManualTransaction 手动管理事务示例
func ExampleManualTransaction() {
	db := database.GetDB()

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("发生恐慌，事务已回滚: %v\n", r)
		}
	}()

	// 操作1: 创建用户
	user := &models.User{
		Name:  "李四",
		Email: "lisi@example.com",
		Age:   30,
	}
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		fmt.Printf("创建用户失败，事务已回滚: %v\n", err)
		return
	}

	// 操作2: 创建订单
	order := &models.Order{
		UserID:  user.ID,
		Product: "MacBook Pro",
		Amount:  14999.00,
		Status:  1,
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		fmt.Printf("创建订单失败，事务已回滚: %v\n", err)
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("提交事务失败: %v\n", err)
		return
	}

	fmt.Println("手动事务成功")
}

// ExampleNestedTransaction 嵌套事务示例
func ExampleNestedTransaction(
	userRepo interfaces.UserRepository,
	orderRepo interfaces.OrderRepository,
	productRepo *repository.ProductRepositoryImpl,
) {
	ctx := context.Background()
	db := database.GetDB()

	// 外层事务
	err := db.Transaction(func(tx *gorm.DB) error {
		// 操作1: 创建产品
		product := &models.Product{
			Name:        "AirPods Pro",
			Description: "苹果无线耳机",
			Price:       1999.00,
			Stock:       100,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}

		// 操作2: 创建用户
		user := &models.User{
			Name:  "赵六",
			Email: "zhaoliu@example.com",
			Age:   35,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			return err
		}

		// 操作3: 创建订单（使用刚创建的产品）
		order := &models.Order{
			UserID:  user.ID,
			Product: product.Name,
			Amount:  product.Price,
			Status:  1,
		}
		if err := orderRepo.Create(ctx, order); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("嵌套事务失败: %v\n", err)
		return
	}

	fmt.Println("嵌套事务成功")
}

// ExampleBatchTransaction 批量操作事务示例
func ExampleBatchTransaction(orderRepo interfaces.OrderRepository) {
	//ctx := context.Background()
	db := database.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		// 批量创建订单
		var orders []*models.Order
		for i := 1; i <= 10; i++ {
			orders = append(orders, &models.Order{
				UserID:  uint(i),
				Product: fmt.Sprintf("商品 %d", i),
				Amount:  float64(i * 100),
				Status:  0,
			})
		}

		// 批量插入（更高效）
		if err := tx.CreateInBatches(orders, 5).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("批量操作事务失败: %v\n", err)
		return
	}

	fmt.Println("批量操作事务成功")
}

// ExampleSavepoint 使用保存点的复杂事务示例
func ExampleSavepoint() {
	db := database.GetDB()

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 操作1: 创建用户
	user := &models.User{
		Name:  "孙七",
		Email: "sunqi@example.com",
		Age:   40,
	}
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		fmt.Printf("操作1失败: %v\n", err)
		return
	}

	// 设置保存点
	savepoint := "after_user_create"
	if err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepoint)).Error; err != nil {
		tx.Rollback()
		fmt.Printf("设置保存点失败: %v\n", err)
		return
	}

	// 操作2: 创建订单（可能失败）
	order := &models.Order{
		UserID:  user.ID,
		Product: "Apple Watch",
		Amount:  3199.00,
		Status:  1,
	}
	if err := tx.Create(order).Error; err != nil {
		// 回滚到保存点，保留用户创建
		tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepoint))
		fmt.Printf("操作2失败，已回滚到保存点: %v\n", err)
		// 可以选择继续或完全回滚
		tx.Rollback()
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("提交失败: %v\n", err)
		return
	}

	fmt.Println("带保存点的事务成功")
}
