package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"go-wework-svc/internal/adapter/client"
	handler "go-wework-svc/internal/adapter/http"
	"go-wework-svc/internal/shared"
	"go-wework-svc/internal/wework"
)

// App 应用程序，组装所有组件
type App struct {
	server *http.Server
	logger *slog.Logger
}

// NewApp 初始化应用：slog logger → Crypto → AIClient → WeWork Service → HTTP Handler → 路由
func NewApp(cfg *shared.Config) (*App, error) {
	logger := initLogger(cfg.Log)

	crypto, err := wework.NewCrypto(cfg.WeWork.Token, cfg.WeWork.EncodingAESKey, cfg.WeWork.CorpID)
	if err != nil {
		return nil, fmt.Errorf("init crypto: %w", err)
	}

	aiClient := client.NewAIClient(cfg.AI, logger)

	wwSvc := wework.NewService(crypto, aiClient, logger)

	callbackHandler := handler.NewCallbackHandler(wwSvc, logger)
	healthHandler := handler.NewHealthHandler()

	mux := http.NewServeMux()
	mux.Handle("/callback", callbackHandler)
	mux.Handle("/health", healthHandler)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{server: server, logger: logger}, nil
}

// Run 启动 HTTP 服务器
func (a *App) Run() error {
	a.logger.Info("starting server", "addr", a.server.Addr)
	return a.server.ListenAndServe()
}

// initLogger 根据配置初始化 slog logger
func initLogger(cfg shared.LogConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(h)
}
