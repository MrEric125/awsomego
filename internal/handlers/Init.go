package handlers

import (
	new2 "awesome/internal/inf/ai/new"
	"awesome/internal/inf/ai/new/service"
	"awesome/internaldig"
	"fmt"
)

func Init() {

	// 5. 注册 AIHandler
	if err := internaldig.Provide(func(aiService service.ChatService) *new2.ChatHandler {
		return new2.NewChatHandler(&aiService)
	}); err != nil {
		fmt.Println("Error providing AIHandler:", err)
		panic(err)
	}
}
