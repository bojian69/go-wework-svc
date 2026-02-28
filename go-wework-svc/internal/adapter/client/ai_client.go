package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go-wework-svc/internal/ai"
	"go-wework-svc/internal/shared"
)

// AIClient AI 助手 HTTP 客户端
type AIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	retry      int
}

// NewAIClient 创建 AI HTTP 客户端
func NewAIClient(cfg shared.AIConfig, logger *slog.Logger) *AIClient {
	return &AIClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
		retry:  cfg.Retry,
	}
}

// SendMessage 实现 ai.Service 接口，将消息发送给 AI 助手
func (c *AIClient) SendMessage(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	var lastErr error
	attempts := c.retry + 1 // first attempt + retries

	for i := range attempts {
		resp, err := c.doRequest(ctx, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// Don't sleep after the last attempt
		if i < c.retry {
			delay := time.Duration(500<<uint(i)) * time.Millisecond // 500ms, 1s, 2s, ...
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	c.logger.Error("all retries failed for AI request",
		"user_id", req.UserID,
		"attempts", attempts,
		"error", lastErr,
	)
	return nil, fmt.Errorf("send message after %d attempts: %w", attempts, lastErr)
}

// doRequest 执行单次 HTTP POST 请求
func (c *AIClient) doRequest(ctx context.Context, body []byte) (*ai.ChatResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ai.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &chatResp, nil
}
