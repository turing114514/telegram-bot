package api

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// IdentityPayload 通用身份载荷字段（与后端 channel_identity.go request 字段对齐）
type IdentityPayload struct {
	ChannelUserID  string `json:"channel_user_id"`
	TelegramUserID string `json:"telegram_user_id"`
	Username       string `json:"username"`
	TelegramUser   string `json:"telegram_username"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	AvatarURL      string `json:"avatar_url"`
	Locale         string `json:"locale"`
}

// BindPayload 邮箱验证码绑定额外字段
type BindPayload struct {
	IdentityPayload
	BindMode string `json:"bind_mode"`
	Email    string `json:"email"`
	Code     string `json:"code"`
}

// IdentityUser 响应的 user 块
type IdentityUser struct {
	ID                    uint   `json:"id"`
	Email                 string `json:"email"`
	DisplayName           string `json:"display_name"`
	Status                string `json:"status"`
	Locale                string `json:"locale"`
	EmailVerified         bool   `json:"email_verified"`
	PasswordSetupRequired bool   `json:"password_setup_required"`
}

// IdentityIdentity 响应的 identity 块
type IdentityIdentity struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
}

// IdentityResponse POST /api/v1/channel/identities/telegram/{resolve,provision,bind}
type IdentityResponse struct {
	StatusCode    int              `json:"status_code"`
	Msg           string           `json:"msg"`
	Data          IdentityData     `json:"data"`
}

// IdentityData 响应 data
type IdentityData struct {
	Bound   bool             `json:"bound"`
	Created bool             `json:"created"`
	Identity *IdentityIdentity `json:"identity,omitempty"`
	User     *IdentityUser    `json:"user,omitempty"`
}

// ResolveTelegramIdentity POST /api/v1/channel/identities/telegram/resolve
func (c *Client) ResolveTelegramIdentity(ctx context.Context, p IdentityPayload) (*IdentityData, error) {
	var resp IdentityResponse
	if err := c.do(ctx, "POST", "/api/v1/channel/identities/telegram/resolve", nil, p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ProvisionTelegramIdentity POST /api/v1/channel/identities/telegram/provision
func (c *Client) ProvisionTelegramIdentity(ctx context.Context, p IdentityPayload) (*IdentityData, error) {
	var resp IdentityResponse
	if err := c.do(ctx, "POST", "/api/v1/channel/identities/telegram/provision", nil, p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// BindTelegramIdentity POST /api/v1/channel/identities/telegram/bind
func (c *Client) BindTelegramIdentity(ctx context.Context, p BindPayload) (*IdentityData, error) {
	var resp IdentityResponse
	if err := c.do(ctx, "POST", "/api/v1/channel/identities/telegram/bind", nil, p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ChannelUserIDFromQuery 工具函数（与 channel_order.go channelUserIDValue 等价）
func ChannelUserIDFromQuery(q url.Values) string {
	if v := q.Get("channel_user_id"); v != "" {
		return v
	}
	return q.Get("telegram_user_id")
}

// ParseUint64 工具
func ParseUint64(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n
}
