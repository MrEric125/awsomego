package examples

import (
	"context"
	"fmt"
	"time"

	"awesome/internal/database"
	"awesome/internal/models"
	"awesome/internal/repository"

	"gorm.io/gorm"
)

// TransactionExamples 工业级事务使用示例
type TransactionExamples struct {
	userRepo    *repository.UserRepositoryImpl
	orderRepo   *repository.OrderRepositoryImpl
	productRepo *repository.ProductRepositoryImpl
	db          *gorm.DB
}

// NewTransactionExamples 创建示例实例
func NewTransactionExamples(db *gorm.DB) *TransactionExamples {
	return &TransactionExamples{
		userRepo:    repository.NewUserRepository(db).(*repository.UserRepositoryImpl),
		orderRepo:   repository.NewOrderRepository(db).(*repository.OrderRepositoryImpl),
		productRepo: repository.NewProductRepository(db),
		db:          db,
	}
}

// Example1_BasicTransaction 基础事务示例
func (e *TransactionExamples) Example1_BasicTransaction() error {
	ctx := context.Background()

	// 使用默认配置
	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 创建用户
		user := &models.User{
			Name:  "张三",
			Email: "zhangsan@example.com",
			Age:   25,
		}
		if err := e.userRepo.Create(ctx, user); err != nil {
			return err
		}

		// 创建订单
		order := &models.Order{
			UserID:  user.ID,
			Product: "iPhone 15",
			Amount:  7999.00,
			Status:  0,
		}
		if err := e.orderRepo.Create(ctx, order); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Println("基础事务执行成功")
	return nil
}

// Example2_CustomTransactionOptions 自定义事务配置
func (e *TransactionExamples) Example2_CustomTransactionOptions() error {
	ctx := context.Background()

	// 自定义事务选项
	opts := database.TransactionOptions{
		Timeout:        5 * time.Second,              // 5秒超时
		IsolationLevel: database.ReadCommitted,       // 读已提交
		RetryCount:     3,                            // 最多重试3次
		RetryDelay:     100 * time.Millisecond,       // 重试间隔100ms
		ReadOnly:       false,                        // 读写事务
		SkipLogging:    false,                        // 记录日志
	}

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		user := &models.User{
			Name:  "李四",
			Email: "lisi@example.com",
			Age:   30,
		}
		return e.userRepo.Create(ctx, user)
	}, opts)

	return err
}

// Example3_HighConsistencyTransaction 高一致性事务（金融场景）
func (e *TransactionExamples) Example3_HighConsistencyTransaction() error {
	ctx := context.Background()

	// 金融级别配置：最高隔离级别，更多重试
	opts := database.TransactionOptions{
		Timeout:        10 * time.Second,
		IsolationLevel: database.Serializable, // 串行化隔离级别
		RetryCount:     5,                     // 更多重试次数
		RetryDelay:     200 * time.Millisecond,
	}

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 转账操作：需要最高一致性保证
		fromUserID := 1
		toUserID := 2
		amount := 100.00

		// 获取转出用户（加锁）
		fromUser, err := e.userRepo.GetByID(ctx, fromUserID)
		if err != nil {
			return err
		}

		// 获取转入用户（加锁）
		toUser, err := e.userRepo.GetByID(ctx, toUserID)
		if err != nil {
			return err
		}

		// 验证余额
		if fromUser.Age < int(amount) {
			return fmt.Errorf("insufficient balance")
		}

		// 执行转账逻辑...
		fmt.Printf("从 %s 转账 %.2f 到 %s\n", fromUser.Name, amount, toUser.Name)

		return nil
	}, opts)

	return err
}

// Example4_ReadOnlyTransaction 只读事务（性能优化）
func (e *TransactionExamples) Example4_ReadOnlyTransaction() error {
	ctx := context.Background()

	// 只读事务：数据库可以优化性能
	opts := database.TransactionOptions{
		ReadOnly: true,
	}

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 批量查询操作
		users, total, err := e.userRepo.ListUsers(ctx, 1, 10)
		if err != nil {
			return err
		}

		fmt.Printf("查询到 %d 个用户，总数: %d\n", len(users), total)
		return nil
	}, opts)

	return err
}

// Example5_NestedTransaction 嵌套事务示例
func (e *TransactionExamples) Example5_NestedTransaction() error {
	ctx := context.Background()

	// 外层事务
	err := database.WithTransaction(ctx, e.db, func(outerCtx context.Context) error {
		// 外层操作：创建用户
		user := &models.User{
			Name:  "王五",
			Email: "wangwu@example.com",
			Age:   28,
		}
		if err := e.userRepo.Create(outerCtx, user); err != nil {
			return err
		}

		// 内层事务：会自动使用外层事务
		err := database.WithTransaction(outerCtx, e.db, func(innerCtx context.Context) error {
			// 内层操作：创建订单
			order := &models.Order{
				UserID:  user.ID,
				Product: "MacBook Pro",
				Amount:  15999.00,
				Status:  1,
			}
			return e.orderRepo.Create(innerCtx, order)
		})

		if err != nil {
			return err
		}

		fmt.Println("嵌套事务执行成功")
		return nil
	})

	return err
}

// Example6_BatchOperation 批量操作事务
func (e *TransactionExamples) Example6_BatchOperation() error {
	ctx := context.Background()

	// 批量操作可能需要更长的超时时间
	opts := database.TransactionOptions{
		Timeout:    60 * time.Second,
		RetryCount: 2,
	}

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 批量创建用户
		users := []models.User{
			{Name: "用户1", Email: "user1@example.com", Age: 20},
			{Name: "用户2", Email: "user2@example.com", Age: 21},
			{Name: "用户3", Email: "user3@example.com", Age: 22},
			{Name: "用户4", Email: "user4@example.com", Age: 23},
			{Name: "用户5", Email: "user5@example.com", Age: 24},
		}

		for i := range users {
			if err := e.userRepo.Create(ctx, &users[i]); err != nil {
				return fmt.Errorf("failed to create user %d: %w", i+1, err)
			}
		}

		fmt.Printf("批量创建了 %d 个用户\n", len(users))
		return nil
	}, opts)

	return err
}

// Example7_OptimisticLocking 乐观锁示例
func (e *TransactionExamples) Example7_OptimisticLocking() error {
	ctx := context.Background()

	opts := database.TransactionOptions{
		RetryCount: 3, // 乐观锁冲突时重试
	}

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 获取产品
		product, err := e.productRepo.GetByID(ctx, 1)
		if err != nil {
			return err
		}

		// 检查库存
		if product.Stock < 1 {
			return fmt.Errorf("库存不足")
		}

		// 原子操作减少库存（乐观锁）
		if err := e.productRepo.DecreaseStock(ctx, 1, 1); err != nil {
			return err
		}

		fmt.Printf("成功购买产品: %s，剩余库存: %d\n", product.Name, product.Stock-1)
		return nil
	}, opts)

	return err
}

// Example8_ErrorHandling 错误处理和回滚
func (e *TransactionExamples) Example8_ErrorHandling() error {
	ctx := context.Background()

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 第一步：创建用户
		user := &models.User{
			Name:  "测试用户",
			Email: "test@example.com",
			Age:   25,
		}
		if err := e.userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("step 1 failed: %w", err)
		}

		// 第二步：模拟业务错误
		if user.Name == "测试用户" {
			// 返回错误会触发自动回滚
			return fmt.Errorf("business validation failed: 用户名不合法")
		}

		// 这行代码不会执行
		order := &models.Order{
			UserID: user.ID,
			Product: "Test",
			Amount:  100,
		}
		return e.orderRepo.Create(ctx, order)
	})

	if err != nil {
		// 事务已回滚，用户不会被创建
		fmt.Printf("事务失败，已回滚: %v\n", err)
		return err
	}

	return nil
}

// Example9_DeadlockRetry 死锁重试示例
func (e *TransactionExamples) Example9_DeadlockRetry() error {
	ctx := context.Background()

	// 配置自动重试死锁
	opts := database.TransactionOptions{
		RetryCount: 3,
		RetryDelay: 100 * time.Millisecond,
	}

	// 并发执行两个可能死锁的事务
	errChan := make(chan error, 2)

	go func() {
		err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
			// 事务1：先更新用户1，再更新用户2
			e.userRepo.UpdateName(ctx, 1, "用户1_更新")
			time.Sleep(10 * time.Millisecond) // 模拟业务处理
			return e.userRepo.UpdateName(ctx, 2, "用户2_更新")
		}, opts)
		errChan <- err
	}()

	go func() {
		err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
			// 事务2：先更新用户2，再更新用户1（可能死锁）
			e.userRepo.UpdateName(ctx, 2, "用户2_更新")
			time.Sleep(10 * time.Millisecond)
			return e.userRepo.UpdateName(ctx, 1, "用户1_更新")
		}, opts)
		errChan <- err
	}()

	// 等待两个事务完成
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			fmt.Printf("事务执行失败: %v\n", err)
		}
	}

	fmt.Println("死锁重试示例完成")
	return nil
}

// Example10_PerformanceMonitoring 性能监控示例
func (e *TransactionExamples) Example10_PerformanceMonitoring() error {
	ctx := context.Background()

	startTime := time.Now()

	err := database.WithTransaction(ctx, e.db, func(ctx context.Context) error {
		// 执行业务逻辑
		user := &models.User{
			Name:  "性能测试",
			Email: "perf@example.com",
			Age:   25,
		}
		return e.userRepo.Create(ctx, user)
	})

	duration := time.Since(startTime)
	fmt.Printf("事务执行耗时: %v, 错误: %v\n", duration, err)

	// 可以根据耗时进行监控告警
	if duration > 5*time.Second {
		fmt.Println("警告：事务执行时间过长")
	}

	return err
}

// RunAllExamples 运行所有示例
func (e *TransactionExamples) RunAllExamples() {
	examples := []struct {
		name string
		fn   func() error
	}{
		{"基础事务", e.Example1_BasicTransaction},
		{"自定义配置", e.Example2_CustomTransactionOptions},
		{"高一致性事务", e.Example3_HighConsistencyTransaction},
		{"只读事务", e.Example4_ReadOnlyTransaction},
		{"嵌套事务", e.Example5_NestedTransaction},
		{"批量操作", e.Example6_BatchOperation},
		{"乐观锁", e.Example7_OptimisticLocking},
		{"错误处理", e.Example8_ErrorHandling},
		{"死锁重试", e.Example9_DeadlockRetry},
		{"性能监控", e.Example10_PerformanceMonitoring},
	}

	for _, example := range examples {
		fmt.Printf("\n=== 示例: %s ===\n", example.name)
		if err := example.fn(); err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println("✓ 成功")
		}
	}
}
