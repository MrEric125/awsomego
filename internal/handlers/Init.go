package handlers

import (
	"awesome/internal/inf/ai/new/service"
	"awesome/internaldig"
	"fmt"
)

func Init() {

	// 5. 注册 AIHandler
	if err := internaldig.Provide(func(aiService *service.ChatService) *ChatHandler {
		return NewChatHandler(aiService)
	}); err != nil {
		fmt.Println("Error providing AIHandler:", err)
		panic(err)
	}
}
