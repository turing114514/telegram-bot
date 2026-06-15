// Package config 负责加载 telegram-bot 自身的 YAML 运行时配置。
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 顶层配置
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	API     APIConfig     `yaml:"api"`
	Channel ChannelConfig `yaml:"channel"`
	Bot     BotConfig     `yaml:"bot"`
	Log     LogConfig     `yaml:"log"`
}

// ServerConfig HTTP 回调服务配置
type ServerConfig struct {
	Listen        string `yaml:"listen"`
	PublicBaseURL string `yaml:"public_base_url"`
	MaxBodyBytes  int64  `yaml:"max_body_bytes"`
}

// APIConfig dujiao-next Channel API 配置
type APIConfig struct {
	BaseURL             string `yaml:"base_url"`
	TimeoutSeconds      int    `yaml:"timeout_seconds"`
	InsecureSkipVerify  bool   `yaml:"insecure_skip_verify"`
	MaxIdleConnsPerHost int    `yaml:"max_idle_conns_per_host"`
}

// ChannelConfig 渠道客户端凭证（敏感，建议优先用环境变量）
type ChannelConfig struct {
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
}

// BotConfig Telegram Bot 自身配置
type BotConfig struct {
	MachineCode         string `yaml:"machine_code"`
	BotVersion          string `yaml:"bot_version"`
	WebhookStatus       string `yaml:"webhook_status"`
	LicenseStatus       string `yaml:"license_status"`
	LicenseExpiresAt    string `yaml:"license_expires_at"`
	DefaultLocale       string `yaml:"default_locale"`
	DefaultCurrency     string `yaml:"default_currency"`
	UpdateIntervalSec   int    `yaml:"update_interval_seconds"`
	ConfigReloadSeconds int    `yaml:"config_reload_seconds"`
	PollTimeoutSeconds  int    `yaml:"poll_timeout_seconds"`
	PollLimit           int    `yaml:"poll_limit"`
	ProductPageSize     int    `yaml:"product_page_size"`
	OrderPageSize       int    `yaml:"order_page_size"`
	WalletPageSize      int    `yaml:"wallet_page_size"`
	AffiliatePageSize   int    `yaml:"affiliate_page_size"`
	QuantityOptions     []int  `yaml:"quantity_options"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `yaml:"level"`
}

// Load 从指定路径加载 YAML，并合并来自环境变量的敏感配置。
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyEnvOverrides(cfg)
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("TG_CHANNEL_KEY")); v != "" {
		cfg.Channel.Key = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_CHANNEL_SECRET")); v != "" {
		cfg.Channel.Secret = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_API_BASE_URL")); v != "" {
		cfg.API.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_BOT_VERSION")); v != "" {
		cfg.Bot.BotVersion = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_MACHINE_CODE")); v != "" {
		cfg.Bot.MachineCode = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_DEFAULT_LOCALE")); v != "" {
		cfg.Bot.DefaultLocale = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_DEFAULT_CURRENCY")); v != "" {
		cfg.Bot.DefaultCurrency = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_LISTEN")); v != "" {
		cfg.Server.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_LICENSE_STATUS")); v != "" {
		cfg.Bot.LicenseStatus = v
	}
	if v := strings.TrimSpace(os.Getenv("TG_LICENSE_EXPIRES_AT")); v != "" {
		cfg.Bot.LicenseExpiresAt = v
	}
}

func (c *Config) normalize() error {
	c.Server.Listen = strings.TrimSpace(c.Server.Listen)
	if c.Server.Listen == "" {
		c.Server.Listen = ":8444"
	}
	if c.Server.MaxBodyBytes <= 0 {
		c.Server.MaxBodyBytes = 1 << 20 // 1MB
	}
	c.API.BaseURL = strings.TrimRight(strings.TrimSpace(c.API.BaseURL), "/")
	if c.API.BaseURL == "" {
		return fmt.Errorf("api.base_url is required")
	}
	if c.API.TimeoutSeconds <= 0 {
		c.API.TimeoutSeconds = 15
	}
	if c.API.MaxIdleConnsPerHost <= 0 {
		c.API.MaxIdleConnsPerHost = 16
	}
	c.Bot.DefaultLocale = strings.TrimSpace(c.Bot.DefaultLocale)
	switch c.Bot.DefaultLocale {
	case "zh-CN", "zh-TW", "en-US":
	default:
		c.Bot.DefaultLocale = "zh-CN"
	}
	c.Bot.DefaultCurrency = strings.TrimSpace(c.Bot.DefaultCurrency)
	if c.Bot.DefaultCurrency == "" {
		c.Bot.DefaultCurrency = "CNY"
	}
	if c.Bot.UpdateIntervalSec <= 0 {
		c.Bot.UpdateIntervalSec = 25
	}
	if c.Bot.UpdateIntervalSec > 55 {
		c.Bot.UpdateIntervalSec = 55 // 必须 < 60s
	}
	if c.Bot.ConfigReloadSeconds <= 0 {
		c.Bot.ConfigReloadSeconds = 60
	}
	if c.Bot.PollTimeoutSeconds <= 0 {
		c.Bot.PollTimeoutSeconds = 10
	}
	if c.Bot.PollLimit <= 0 {
		c.Bot.PollLimit = 100
	}
	if c.Bot.WebhookStatus == "" {
		c.Bot.WebhookStatus = "active"
	}
	if c.Bot.LicenseStatus == "" {
		c.Bot.LicenseStatus = "active"
	}
	if c.Bot.ProductPageSize <= 0 {
		c.Bot.ProductPageSize = 5
	}
	if c.Bot.OrderPageSize <= 0 {
		c.Bot.OrderPageSize = 5
	}
	if c.Bot.WalletPageSize <= 0 {
		c.Bot.WalletPageSize = 10
	}
	if c.Bot.AffiliatePageSize <= 0 {
		c.Bot.AffiliatePageSize = 8
	}
	if len(c.Bot.QuantityOptions) == 0 {
		c.Bot.QuantityOptions = []int{1, 2, 5, 10}
	}
	return nil
}

// ChannelKey 返回渠道客户端 key（YAML → 环境变量）
func (c *Config) ChannelKey() string { return strings.TrimSpace(c.Channel.Key) }

// ChannelSecret 返回渠道客户端 secret（YAML → 环境变量）
func (c *Config) ChannelSecret() string { return strings.TrimSpace(c.Channel.Secret) }

// APIBaseURL 对外暴露给 cmd 使用的 API base
func (c *Config) APIBaseURL() string { return c.API.BaseURL }

// UpdateInterval 心跳周期
func (c *Config) UpdateInterval() time.Duration {
	return time.Duration(c.Bot.UpdateIntervalSec) * time.Second
}

// ReloadInterval 配置重拉周期
func (c *Config) ReloadInterval() time.Duration {
	return time.Duration(c.Bot.ConfigReloadSeconds) * time.Second
}

// APIRequestTimeout API 单次请求超时
func (c *Config) APIRequestTimeout() time.Duration {
	return time.Duration(c.API.TimeoutSeconds) * time.Second
}

// QuantityQuickOptions 商品快速选择数量按钮
func (c *Config) QuantityQuickOptions() []int {
	return c.Bot.QuantityOptions
}
