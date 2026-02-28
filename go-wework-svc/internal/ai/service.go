package ai

import "context"

// Service AI 助手服务接口
type Service interface {
	// SendMessage 将消息发送给 AI 助手
	SendMessage(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
