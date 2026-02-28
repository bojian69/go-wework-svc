package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"go-wework-svc/internal/wework"
)

// CallbackHandler 企业微信回调 HTTP 处理器
type CallbackHandler struct {
	svc    wework.Service
	logger *slog.Logger
}

// NewCallbackHandler 创建回调处理器实例
func NewCallbackHandler(svc wework.Service, logger *slog.Logger) *CallbackHandler {
	return &CallbackHandler{svc: svc, logger: logger}
}

// ServeHTTP 统一处理 GET（URL 验证）和 POST（消息回调）请求
func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerifyURL(w, r)
	case http.MethodPost:
		h.handleCallback(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVerifyURL 处理 GET 请求的 URL 验证
func (h *CallbackHandler) handleVerifyURL(w http.ResponseWriter, r *http.Request) {
	q := wework.CallbackQuery{
		MsgSignature: r.URL.Query().Get("msg_signature"),
		Timestamp:    r.URL.Query().Get("timestamp"),
		Nonce:        r.URL.Query().Get("nonce"),
		Echostr:      r.URL.Query().Get("echostr"),
	}

	plaintext, err := h.svc.VerifyURL(r.Context(), q)
	if err != nil {
		if errors.Is(err, wework.ErrInvalidSignature) {
			h.logger.Warn("URL verification signature failed",
				"timestamp", q.Timestamp,
				"nonce", q.Nonce,
			)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h.logger.Error("URL verification failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(plaintext))
}

// handleCallback 处理 POST 请求的消息回调
func (h *CallbackHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	q := wework.CallbackQuery{
		MsgSignature: r.URL.Query().Get("msg_signature"),
		Timestamp:    r.URL.Query().Get("timestamp"),
		Nonce:        r.URL.Query().Get("nonce"),
	}

	err = h.svc.HandleCallback(r.Context(), q, body)
	if err != nil {
		if strings.Contains(err.Error(), "unmarshal") {
			h.logger.Warn("callback XML parse failed", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if errors.Is(err, wework.ErrInvalidSignature) {
			h.logger.Warn("callback signature failed",
				"timestamp", q.Timestamp,
				"nonce", q.Nonce,
			)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h.logger.Error("callback processing failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}
