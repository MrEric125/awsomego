package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement;comment:用户ID" json:"id"`
	Name      string         `gorm:"size:100;not null;comment:用户名" json:"name"`
	Email     string         `gorm:"size:255;uniqueIndex;comment:邮箱" json:"email"`
	Age       int            `gorm:"comment:年龄" json:"age"`
	CreatedAt time.Time      `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time      `gorm:"comment:更新时间" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"comment:删除时间" json:"deleted_at,omitempty"`

	// 关联关系
	Orders []Order `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"orders,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// Order 订单模型
type Order struct {
	ID        uint           `gorm:"primaryKey;autoIncrement;comment:订单ID" json:"id"`
	UserID    uint           `gorm:"not null;index;comment:用户ID" json:"user_id"`
	Product   string         `gorm:"size:255;not null;comment:商品名称" json:"product"`
	Amount    float64        `gorm:"type:decimal(10,2);not null;comment:金额" json:"amount"`
	Status    int            `gorm:"type:tinyint;default:0;comment:订单状态 0:待支付 1:已支付 2:已取消" json:"status"`
	CreatedAt time.Time      `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time      `gorm:"comment:更新时间" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"comment:删除时间" json:"deleted_at,omitempty"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Order) TableName() string {
	return "orders"
}

// Product 产品模型（演示多表）
type Product struct {
	ID          uint           `gorm:"primaryKey;autoIncrement;comment:产品ID" json:"id"`
	Name        string         `gorm:"size:255;not null;comment:产品名称" json:"name"`
	Description string         `gorm:"type:text;comment:产品描述" json:"description"`
	Price       float64        `gorm:"type:decimal(10,2);not null;comment:价格" json:"price"`
	Stock       int            `gorm:"default:0;comment:库存" json:"stock"`
	CreatedAt   time.Time      `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"comment:更新时间" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"comment:删除时间" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (Product) TableName() string {
	return "products"
}
