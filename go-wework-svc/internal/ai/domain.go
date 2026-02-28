package ai

// ChatRequest AI 助手请求
type ChatRequest struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
	Source  string `json:"source"` // "wework"
	GroupID string `json:"group_id,omitempty"`
}

// ChatResponse AI 助手响应
type ChatResponse struct {
	Reply string `json:"reply"`
}
