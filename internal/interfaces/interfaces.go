package interfaces

import (
	"context"
	"gorm.io/gorm"

	"awesome/internal/models"
)

// Repository 接口定义了数据存储的行为
type Repository interface {
	Create(ctx context.Context, model interface{}) error
	Update(ctx context.Context, model interface{}) error
	Delete(ctx context.Context, model interface{}, id interface{}) error
	FindByID(ctx context.Context, model interface{}, id interface{}) error
	Find(ctx context.Context, model interface{}, query interface{}, args ...interface{}) error
	Count(ctx context.Context, model interface{}, query interface{}, args ...interface{}) (int64, error)
	Exec(ctx context.Context, sql string, values ...interface{}) error
	Raw(ctx context.Context, sql string, values ...interface{}) (*gorm.DB, error)
}

// UserService 接口定义了用户服务的行为
type UserService interface {
	SGetUserName(id int) string
	CreateUserWithOrder(name, email string, age int, productName string, amount float64) error
	CreateComplexOrder(userID int, productID int, quantity int) error
	TransferUsers(fromUserID, toUserID int, amount float64) error
	BatchCreateUsers(users []models.User) error
}

type AnotherService interface {
	DoSomething() string
}

type UserRepository interface {

	// 显式传入 db 实例，可能是事务对象，也可能是普通连接
	GetByID(ctx context.Context, id int) (*models.User, error)
	UpdateName(ctx context.Context, id int, name string) error
	Create(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
	ListUsers(ctx context.Context, page, pageSize int) ([]models.User, int64, error)
}

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id int) (*models.Order, error)
	GetByUserID(ctx context.Context, userID int, page, pageSize int) ([]models.Order, int64, error)
	UpdateStatus(ctx context.Context, id int, status int) error
	Delete(ctx context.Context, id int) error
	BatchCreate(ctx context.Context, orders []*models.Order) error
}
