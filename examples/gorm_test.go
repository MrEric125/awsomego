package main

import (
	"fmt"
	"log"
	"testing"

	_ "awesome/internal"
	"awesome/internal/database"
	"awesome/internal/interfaces"
	"awesome/internal/models"
	"awesome/internaldig"
)

func TestGormTest(t *testing.T) {
	fmt.Println("=== GORM 事务管理示例 ===")

	// 获取服务实例
	userService, err := internaldig.Get[interfaces.UserService]()
	if err != nil {
		log.Fatalf("获取 UserService 失败: %v", err)
	}

	// 示例1: 创建用户并创建订单（简单事务）
	fmt.Println("【示例1】创建用户并创建订单")
	err = userService.CreateUserWithOrder("测试用户", "test@example.com", 25, "iPhone 15", 7999.00)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✓ 成功创建用户和订单")
	}

	// 示例2: 复杂订单创建（涉及库存管理）
	fmt.Println("【示例2】创建复杂订单（需要先检查产品和库存）")
	// 首先创建一个产品
	db := database.GetDB()
	product := &models.Product{
		Name:        "MacBook Pro",
		Description: "苹果笔记本电脑",
		Price:       14999.00,
		Stock:       50,
	}
	if err := db.Create(product).Error; err != nil {
		fmt.Printf("创建产品失败: %v\n", err)
	} else {
		fmt.Printf("✓ 创建产品成功，ID: %d\n\n", product.ID)

		// 然后创建订单（会减少库存）
		err = userService.CreateComplexOrder(1, int(product.ID), 2)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println("✓ 成功创建复杂订单")
		}
	}

	// 示例3: 批量创建用户
	fmt.Println("【示例3】批量创建用户")
	var users []models.User
	for i := 1; i <= 5; i++ {
		users = append(users, models.User{
			Name:  fmt.Sprintf("用户%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i,
		})
	}
	err = userService.BatchCreateUsers(users)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✓ 批量创建用户成功")
	}

	// 示例4: 用户间转账
	fmt.Println("【示例4】用户间转账")
	err = userService.TransferUsers(1, 2, 100.00)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✓ 转账成功")
	}

	fmt.Println("=== 所有示例完成 ===")
}
