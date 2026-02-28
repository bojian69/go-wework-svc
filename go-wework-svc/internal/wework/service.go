package wework

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"

	"go-wework-svc/internal/ai"
)

// Service 企业微信领域服务接口
type Service interface {
	// VerifyURL 处理 GET 请求的 URL 验证
	VerifyURL(ctx context.Context, q CallbackQuery) (string, error)

	// HandleCallback 处理 POST 请求的消息回调
	HandleCallback(ctx context.Context, q CallbackQuery, body []byte) error
}

// serviceImpl Service 接口的实现
type serviceImpl struct {
	crypto Crypto
	aiSvc  ai.Service
	logger *slog.Logger
}

// NewService 创建企业微信领域服务实例
func NewService(crypto Crypto, aiSvc ai.Service, logger *slog.Logger) Service {
	return &serviceImpl{
		crypto: crypto,
		aiSvc:  aiSvc,
		logger: logger,
	}
}

// VerifyURL 处理企业微信 URL 验证请求
// 1. 验证签名 2. 解密 echostr 3. 返回明文
func (s *serviceImpl) VerifyURL(ctx context.Context, q CallbackQuery) (string, error) {
	if !s.crypto.VerifySignature(q.MsgSignature, q.Timestamp, q.Nonce, q.Echostr) {
		return "", ErrInvalidSignature
	}

	plaintext, err := s.crypto.Decrypt(q.Echostr)
	if err != nil {
		return "", fmt.Errorf("decrypt echostr: %w", err)
	}

	return string(plaintext), nil
}

// HandleCallback 处理企业微信消息回调
// 1. 解析加密 XML 2. 验证签名 3. 解密 4. 解析明文 XML 5. 检测 @提及 6. 异步转发 AI
func (s *serviceImpl) HandleCallback(ctx context.Context, q CallbackQuery, body []byte) error {
	// 1. 解析加密 XML
	var encBody EncryptedBody
	if err := xml.Unmarshal(body, &encBody); err != nil {
		return fmt.Errorf("unmarshal encrypted body: %w", err)
	}

	// 2. 验证签名
	if !s.crypto.VerifySignature(q.MsgSignature, q.Timestamp, q.Nonce, encBody.Encrypt) {
		s.logger.Warn("signature verification failed",
			"timestamp", q.Timestamp,
			"nonce", q.Nonce,
		)
		return ErrInvalidSignature
	}

	// 3. 解密消息
	plaintext, err := s.crypto.Decrypt(encBody.Encrypt)
	if err != nil {
		s.logger.Error("failed to decrypt message", "error", err)
		return fmt.Errorf("decrypt message: %w", err)
	}

	// 4. 解析明文 XML
	var msg Message
	if err := xml.Unmarshal(plaintext, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	// 5. 仅处理文本消息中的 @提及
	if msg.MsgType != MsgTypeText {
		return nil
	}
	if !containsMention(msg.Content) {
		return nil
	}

	// 6. 异步转发给 AI（不阻塞响应）
	go s.forwardToAI(context.WithoutCancel(ctx), msg)

	return nil
}

// containsMention 检测消息内容是否包含 @提及
func containsMention(content string) bool {
	return strings.Contains(content, "@")
}

// forwardToAI 将 @提及消息异步转发给 AI 助手
func (s *serviceImpl) forwardToAI(ctx context.Context, msg Message) {
	req := ai.ChatRequest{
		UserID:  msg.FromUserName,
		Content: msg.Content,
		Source:  "wework",
	}

	_, err := s.aiSvc.SendMessage(ctx, req)
	if err != nil {
		s.logger.Error("failed to forward message to AI",
			"user_id", msg.FromUserName,
			"error", err,
		)
		return
	}

	s.logger.Info("message forwarded to AI",
		"msg_id", msg.MsgID,
		"from_user", msg.FromUserName,
	)
}
