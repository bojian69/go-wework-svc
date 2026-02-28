package shared

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server ServerConfig `yaml:"server"`
	WeWork WeWorkConfig `yaml:"wework"`
	AI     AIConfig     `yaml:"ai"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// WeWorkConfig 企业微信配置
type WeWorkConfig struct {
	CorpID         string `yaml:"corp_id"`
	Token          string `yaml:"token"`
	EncodingAESKey string `yaml:"encoding_aes_key"`
	AgentID        int64  `yaml:"agent_id"`
}

// AIConfig AI 助手配置
type AIConfig struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
	Retry   int           `yaml:"retry"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

var alphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// LoadConfig 从 YAML 文件加载并验证配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	// server.addr
	if err := validateAddr(c.Server.Addr); err != nil {
		return fmt.Errorf("server.addr: %w", err)
	}

	// wework.corp_id
	if c.WeWork.CorpID == "" {
		return fmt.Errorf("wework.corp_id: must not be empty")
	}

	// wework.token
	if c.WeWork.Token == "" {
		return fmt.Errorf("wework.token: must not be empty")
	}
	if len(c.WeWork.Token) > 32 {
		return fmt.Errorf("wework.token: must be at most 32 characters, got %d", len(c.WeWork.Token))
	}
	if !alphanumericRegex.MatchString(c.WeWork.Token) {
		return fmt.Errorf("wework.token: must contain only alphanumeric characters")
	}

	// wework.encoding_aes_key
	if len(c.WeWork.EncodingAESKey) != 43 {
		return fmt.Errorf("wework.encoding_aes_key: must be exactly 43 characters, got %d", len(c.WeWork.EncodingAESKey))
	}
	if !alphanumericRegex.MatchString(c.WeWork.EncodingAESKey) {
		return fmt.Errorf("wework.encoding_aes_key: must contain only alphanumeric characters")
	}

	// ai.base_url
	if err := validateBaseURL(c.AI.BaseURL); err != nil {
		return fmt.Errorf("ai.base_url: %w", err)
	}

	return nil
}

func validateAddr(addr string) error {
	if addr == "" {
		return fmt.Errorf("must not be empty")
	}
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}
	return nil
}

func validateBaseURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("must not be empty")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("must include scheme and host")
	}
	return nil
}
