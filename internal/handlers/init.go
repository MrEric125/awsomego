package handlers

import (
	"awesome/internal/interfaces"
	"awesome/internaldig"
	"fmt"
)

func Init() {
	// 注册 UserHandler 到依赖注入容器
	if err := internaldig.Provide(func(userService interfaces.UserService) *UserHandler {
		return NewUserHandler(userService)
	}); err != nil {
		fmt.Println("Error providing UserHandler:", err)
		panic(err)
	}
}
