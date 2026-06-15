package api

import (
	"context"
	"net/url"
	"time"
)

// BotConfig 后台下发的完整 Bot 配置（与 internal/service/telegram_bot_setting.go 字段对齐）
type BotConfig struct {
	Enabled       bool                   `json:"enabled"`
	BotToken      string                 `json:"bot_token"`
	DefaultLocale string                 `json:"default_locale"`
	ConfigVersion int                    `json:"config_version"`
	Basic         BotConfigBasic         `json:"basic"`
	Welcome       BotConfigWelcome       `json:"welcome"`
	Help          BotConfigHelp          `json:"help"`
	Menu          BotConfigMenu          `json:"menu"`
}

// LocalizedText 多语言文本
type LocalizedText map[string]string

// BotConfigBasic 基础信息
type BotConfigBasic struct {
	DisplayName string        `json:"display_name"`
	Description LocalizedText `json:"description"`
	SupportURL  string        `json:"support_url"`
	CoverURL    string        `json:"cover_url"`
}

// BotConfigWelcome 欢迎消息
type BotConfigWelcome struct {
	Enabled bool          `json:"enabled"`
	Message LocalizedText `json:"message"`
}

// BotConfigHelp 帮助中心
type BotConfigHelp struct {
	Enabled     bool                  `json:"enabled"`
	Title       LocalizedText         `json:"title"`
	Intro       LocalizedText         `json:"intro"`
	CenterHint  LocalizedText         `json:"center_hint"`
	SupportHint LocalizedText         `json:"support_hint"`
	Items       []BotConfigHelpItem   `json:"items"`
}

// BotConfigHelpItem 单个帮助主题
type BotConfigHelpItem struct {
	Key             string        `json:"key"`
	Enabled         bool          `json:"enabled"`
	Order           int           `json:"order"`
	Summary         LocalizedText `json:"summary"`
	Title           LocalizedText `json:"title"`
	Content         LocalizedText `json:"content"`
	ShowSupportLink bool          `json:"show_support_link"`
}

// BotConfigMenu 菜单
type BotConfigMenu struct {
	Items []BotConfigMenuItem `json:"items"`
}

// BotConfigMenuItem 单个菜单项
type BotConfigMenuItem struct {
	Key     string                `json:"key"`
	Enabled bool                  `json:"enabled"`
	Order   int                   `json:"order"`
	Label   LocalizedText         `json:"label"`
	Action  BotConfigMenuAction   `json:"action"`
}

// BotConfigMenuAction 菜单动作
type BotConfigMenuAction struct {
	Type  string `json:"type"`  // builtin | url | web_app | command
	Value string `json:"value"` // builtin key / url / command
}

// GetBotConfig GET /api/v1/channel/telegram/config
// 返回 config + config_version 两层结构
func (c *Client) GetBotConfig(ctx context.Context) (*BotConfig, int, error) {
	var resp struct {
		StatusCode int       `json:"status_code"`
		Msg        string    `json:"msg"`
		Data       BotConfig `json:"data"`
		ConfigVer  int       `json:"config_version"`
	}
	if err := c.do(ctx, "GET", "/api/v1/channel/telegram/config", nil, nil, &resp); err != nil {
		return nil, 0, err
	}
	cfg := resp.Data
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = resp.ConfigVer
	}
	return &cfg, cfg.ConfigVersion, nil
}

// HeartbeatRequest 心跳请求体
type HeartbeatRequest struct {
	BotVersion       string   `json:"bot_version"`
	WebhookStatus    string   `json:"webhook_status"`
	MachineCode      string   `json:"machine_code"`
	LicenseStatus    string   `json:"license_status"`
	LicenseExpiresAt string   `json:"license_expires_at"`
	Warnings         []string `json:"warnings"`
}

// ReportHeartbeat POST /api/v1/channel/telegram/heartbeat
func (c *Client) ReportHeartbeat(ctx context.Context, req HeartbeatRequest) (int, error) {
	var resp struct {
		StatusCode   int    `json:"status_code"`
		Msg          string `json:"msg"`
		Data         struct{} `json:"data"`
		ConfigVer    int    `json:"config_version"`
	}
	if err := c.do(ctx, "POST", "/api/v1/channel/telegram/heartbeat", nil, req, &resp); err != nil {
		return 0, err
	}
	return resp.ConfigVer, nil
}

// FormatLicenseExpiresAt 工具：把 time.Time 格式化为 RFC3339 字符串
func FormatLicenseExpiresAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// QueryValues 工具：把 url.Values 转 url.Values
func QueryValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	return values
}
