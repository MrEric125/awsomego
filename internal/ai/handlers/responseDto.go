package handlers

// TranslateRequest 翻译请求
type TranslateRequest struct {
	Text       string `json:"text" binding:"required"`
	TargetLang string `json:"target_lang" binding:"required"`
}

// TranslateResponse 翻译响应
type TranslateResponse struct {
	Translation string `json:"translation"`
}

// SummarizeRequest 摘要请求
type SummarizeRequest struct {
	Text string `json:"text" binding:"required"`
}

// SummarizeResponse 摘要响应
type SummarizeResponse struct {
	Summary string `json:"summary"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Message string    `json:"message" binding:"required"`
	History []Message `json:"history,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Reply   string `json:"reply"`
	Created int64  `json:"created"`
}
