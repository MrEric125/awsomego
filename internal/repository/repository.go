package repository

import (
	"awesome/internal/database"
	"awesome/internal/interfaces"
	"awesome/internaldig"
	"fmt"
)

func Init() {
	db := database.GetDB()

	// 注册 UserRepository
	if err := internaldig.DigContainer.Provide(func() interfaces.UserRepository {
		return NewUserRepository(db)
	}); err != nil {
		fmt.Println("Error providing UserRepository:", err)
		panic(err)
	}

	// 注册 OrderRepository
	if err := internaldig.DigContainer.Provide(func() interfaces.OrderRepository {
		return NewOrderRepository(db)
	}); err != nil {
		fmt.Println("Error providing OrderRepository:", err)
		panic(err)
	}

	// 注册 ProductRepository
	if err := internaldig.DigContainer.Provide(func() *ProductRepositoryImpl {
		return NewProductRepository(db)
	}); err != nil {
		fmt.Println("Error providing ProductRepository:", err)
		panic(err)
	}
}
