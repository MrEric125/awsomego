package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// NewGormDB 创建 GORM 数据库连接池
func NewGormDB() *gorm.DB {
	dsn := os.Getenv("DATABASE_DSN") // 从环境变量获取，例如: user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	if dsn == "" {
		// 默认配置，生产环境应该从配置文件读取
		dsn = "root:Root@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=true"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 日志级别: Info, Warn, Error, Silent
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间

	DB = db
	return db
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	if DB == nil {
		log.Fatal("database not initialized")
	}
	return DB
}

// BeginTransaction 开启一个新事务
func BeginTransaction() *gorm.DB {
	return GetDB().Begin()
}

// Transaction 执行带事务的操作（自动管理提交和回滚）
func Transaction(fc func(tx *gorm.DB) error) error {
	return GetDB().Transaction(fc)
}

// HealthCheck 检查数据库连接状态
func HealthCheck() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.Ping()
}
