package database

import (
	"awesome/internal/models"
	"log"

	"gorm.io/gorm"
)

// AutoMigrate 自动创建/更新数据库表结构
func AutoMigrate(db *gorm.DB) {
	log.Println("开始数据库迁移...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Order{},
		&models.Product{},
	)

	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("数据库迁移完成")
}

// InitDatabase 初始化数据库（连接 + 迁移）
func InitDatabase() *gorm.DB {
	db := NewGormDB()
	// AutoMigrate(db)
	return db
}

// CreateSampleData 创建示例数据（仅用于测试）
func CreateSampleData(db *gorm.DB) {
	log.Println("创建示例数据...")

	// 创建示例用户
	users := []models.User{
		{Name: "张三", Email: "zhangsan@example.com", Age: 25},
		{Name: "李四", Email: "lisi@example.com", Age: 30},
		{Name: "王五", Email: "wangwu@example.com", Age: 28},
	}

	for _, user := range users {
		if err := db.FirstOrCreate(&user, models.User{Email: user.Email}).Error; err != nil {
			log.Printf("创建用户失败: %v", err)
		}
	}

	// 创建示例产品
	products := []models.Product{
		{Name: "iPhone 15", Description: "苹果手机", Price: 7999.00, Stock: 100},
		{Name: "MacBook Pro", Description: "苹果笔记本", Price: 14999.00, Stock: 50},
		{Name: "iPad Air", Description: "苹果平板", Price: 4999.00, Stock: 80},
		{Name: "AirPods Pro", Description: "苹果耳机", Price: 1999.00, Stock: 150},
	}

	for _, product := range products {
		if err := db.FirstOrCreate(&product, models.Product{Name: product.Name}).Error; err != nil {
			log.Printf("创建产品失败: %v", err)
		}
	}

	log.Println("示例数据创建完成")
}
