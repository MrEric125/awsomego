package service

import (
	"awesome/internaldig"
	"context"
	"errors"
	"fmt"
	"time"

	"awesome/internal/database"
	"awesome/internal/interfaces"
	"awesome/internal/models"
	"awesome/internal/repository"
)

// UserServiceImpl2 用户服务实现（支持事务）
type UserServiceImpl2 struct {
	userRepo    interfaces.UserRepository
	orderRepo   interfaces.OrderRepository
	productRepo *repository.ProductRepositoryImpl
}

func Init() {
	if err := internaldig.DigContainer.Provide(func(userRepo interfaces.UserRepository,
		orderRepo interfaces.OrderRepository,
		productRepo *repository.ProductRepositoryImpl) interfaces.UserService {
		return NewUserServiceImpl2(userRepo, orderRepo, productRepo)
	}); err != nil {
		fmt.Println("Error providing UserRepository:", err)
		panic(err)
	}

}

// NewUserServiceImpl2 创建用户服务实例
func NewUserServiceImpl2(
	userRepo interfaces.UserRepository,
	orderRepo interfaces.OrderRepository,
	productRepo *repository.ProductRepositoryImpl,
) interfaces.UserService {
	return &UserServiceImpl2{
		userRepo:    userRepo,
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

// SGetUserName 获取用户名（简单查询，不需要事务）
func (s *UserServiceImpl2) SGetUserName(id int) string {
	ctx := context.Background()
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return user.Name
}

// CreateUserWithOrder 创建用户并为其创建订单（事务操作示例）
func (s *UserServiceImpl2) CreateUserWithOrder(name, email string, age int, productName string, amount float64) error {
	ctx := context.Background()

	// 工业级事务：带重试、超时、监控
	opts := database.TransactionOptions{
		Timeout:        10 * time.Second,
		IsolationLevel: database.ReadCommitted,
		RetryCount:     3,
		RetryDelay:     100 * time.Millisecond,
	}

	err := database.WithTransaction(ctx, database.GetDB(), func(ctx context.Context) error {
		// 1. 创建用户
		user := &models.User{
			Name:  name,
			Email: email,
			Age:   age,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// 2. 创建订单
		order := &models.Order{
			UserID:  user.ID,
			Product: productName,
			Amount:  amount,
			Status:  0, // 待支付
		}
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// 模拟错误测试回滚
		if name == "error" {
			return errors.New("模拟错误，触发回滚")
		}

		return nil
	}, opts)

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Printf("成功创建用户和订单 - 订单商品: %s\n", productName)
	return nil
}

// CreateComplexOrder 创建复杂订单（涉及多个表的操作）
func (s *UserServiceImpl2) CreateComplexOrder(userID int, productID int, quantity int) error {
	ctx := context.Background()

	// 工业级事务配置
	opts := database.TransactionOptions{
		Timeout:        15 * time.Second,
		IsolationLevel: database.Serializable, // 高一致性要求
		RetryCount:     3,
		RetryDelay:     200 * time.Millisecond,
	}

	err := database.WithTransaction(ctx, database.GetDB(), func(ctx context.Context) error {
		// 1. 检查用户是否存在
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		// 2. 检查产品是否存在并减少库存
		product, err := s.productRepo.GetByID(ctx, productID)
		if err != nil {
			return fmt.Errorf("product not found: %w", err)
		}

		if product.Stock < quantity {
			return fmt.Errorf("insufficient stock: available %d, requested %d", product.Stock, quantity)
		}

		// 减少库存（原子操作）
		if err := s.productRepo.DecreaseStock(ctx, productID, quantity); err != nil {
			return fmt.Errorf("failed to decrease stock: %w", err)
		}

		// 3. 创建订单
		totalAmount := product.Price * float64(quantity)
		order := &models.Order{
			UserID:  user.ID,
			Product: product.Name,
			Amount:  totalAmount,
			Status:  1, // 已支付
		}
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		fmt.Printf("订单创建成功 - 用户: %s, 产品: %s, 数量: %d, 总金额: %.2f\n",
			user.Name, product.Name, quantity, totalAmount)

		return nil
	}, opts)

	if err != nil {
		return fmt.Errorf("complex order transaction failed: %w", err)
	}

	return nil
}

// TransferUsers 用户间转账示例（演示跨多个实体的事务）
func (s *UserServiceImpl2) TransferUsers(fromUserID, toUserID int, amount float64) error {
	ctx := context.Background()

	// 金融级别事务：最高隔离级别
	opts := database.TransactionOptions{
		Timeout:        10 * time.Second,
		IsolationLevel: database.Serializable,
		RetryCount:     5, // 更多重试次数
		RetryDelay:     150 * time.Millisecond,
	}

	err := database.WithTransaction(ctx, database.GetDB(), func(ctx context.Context) error {
		// 1. 获取两个用户信息（加锁读取）
		fromUser, err := s.userRepo.GetByID(ctx, fromUserID)
		if err != nil {
			return fmt.Errorf("from user not found: %w", err)
		}

		toUser, err := s.userRepo.GetByID(ctx, toUserID)
		if err != nil {
			return fmt.Errorf("to user not found: %w", err)
		}

		// 2. 验证余额
		if float64(fromUser.Age) < amount { // 用 Age 字段模拟余额
			return fmt.Errorf("insufficient balance")
		}

		// 3. 执行转账
		fmt.Printf("从用户 %s 转账 %.2f 到用户 %s\n", fromUser.Name, amount, toUser.Name)

		// 4. 创建转账记录
		transferRecord := map[string]interface{}{
			"from_user_id": fromUserID,
			"to_user_id":   toUserID,
			"amount":       amount,
			"created_at":   time.Now(),
		}
		fmt.Printf("转账记录: %+v\n", transferRecord)

		return nil
	}, opts)

	if err != nil {
		return fmt.Errorf("transfer transaction failed: %w", err)
	}

	return nil
}

// BatchCreateUsers 批量创建用户（批量操作示例）
func (s *UserServiceImpl2) BatchCreateUsers(users []models.User) error {
	ctx := context.Background()

	// 批量操作事务配置
	opts := database.TransactionOptions{
		Timeout:        30 * time.Second, // 批量操作可能需要更长时间
		IsolationLevel: database.ReadCommitted,
		RetryCount:     2,
		RetryDelay:     100 * time.Millisecond,
	}

	err := database.WithTransaction(ctx, database.GetDB(), func(ctx context.Context) error {
		for i := range users {
			if err := s.userRepo.Create(ctx, &users[i]); err != nil {
				return fmt.Errorf("failed to create user %d: %w", i+1, err)
			}
		}
		return nil
	}, opts)

	if err != nil {
		return fmt.Errorf("batch create users failed: %w", err)
	}

	fmt.Printf("成功批量创建 %d 个用户\n", len(users))
	return nil
}
