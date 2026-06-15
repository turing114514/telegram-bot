package api

import (
	"context"
	"net/url"
	"strconv"
)

// AffiliateClickRequest 点击追踪请求
type AffiliateClickRequest struct {
	ChannelUserID  string `json:"channel_user_id,omitempty"`
	TelegramUserID string `json:"telegram_user_id,omitempty"`
	AffiliateCode  string `json:"affiliate_code" binding:"required"`
	VisitorKey     string `json:"visitor_key,omitempty"`
	LandingPath    string `json:"landing_path,omitempty"`
	Referrer       string `json:"referrer,omitempty"`
}

// TrackAffiliateClick POST /api/v1/channel/affiliate/click
func (c *Client) TrackAffiliateClick(ctx context.Context, req AffiliateClickRequest) error {
	var resp Response[map[string]bool]
	return c.do(ctx, "POST", "/api/v1/channel/affiliate/click", nil, req, &resp)
}

// OpenAffiliate POST /api/v1/channel/affiliate/open
func (c *Client) OpenAffiliate(ctx context.Context, ident IdentityPayload) error {
	var resp Response[map[string]any]
	return c.do(ctx, "POST", "/api/v1/channel/affiliate/open", nil, ident, &resp)
}

// AffiliateProfile 推广档案
type AffiliateProfile struct {
	ID     uint   `json:"id"`
	UserID uint   `json:"user_id"`
	Code   string `json:"code"`
	Status string `json:"status"`
}

// OpenAffiliateAndGet 开通并返回档案
func (c *Client) OpenAffiliateAndGet(ctx context.Context, ident IdentityPayload) (*AffiliateProfile, error) {
	var resp Response[AffiliateProfile]
	if err := c.do(ctx, "POST", "/api/v1/channel/affiliate/open", nil, ident, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// AffiliateDashboard 推广概览
type AffiliateDashboard struct {
	Opened              bool   `json:"opened"`
	AffiliateCode       string `json:"affiliate_code"`
	PromotionPath       string `json:"promotion_path"`
	ClickCount          int64  `json:"click_count"`
	ValidOrderCount     int64  `json:"valid_order_count"`
	ConversionRate      string `json:"conversion_rate"`
	PendingCommission   string `json:"pending_commission"`
	AvailableCommission string `json:"available_commission"`
	WithdrawnCommission string `json:"withdrawn_commission"`
	MinWithdrawAmount   string `json:"min_withdraw_amount"`
	WithdrawChannels    []any  `json:"withdraw_channels"`
}

// GetAffiliateDashboard GET /api/v1/channel/affiliate/dashboard
func (c *Client) GetAffiliateDashboard(ctx context.Context, channelUserID string) (*AffiliateDashboard, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	var resp PageResponse[AffiliateDashboard]
	if err := c.do(ctx, "GET", "/api/v1/channel/affiliate/dashboard", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// AffiliateCommission 佣金记录
type AffiliateCommission struct {
	ID                  uint   `json:"id"`
	AffiliateProfileID  uint   `json:"affiliate_profile_id"`
	OrderID             uint   `json:"order_id"`
	OrderNo             string `json:"order_no"`
	CommissionType      string `json:"commission_type"`
	BaseAmount          string `json:"base_amount"`
	RatePercent         string `json:"rate_percent"`
	CommissionAmount    string `json:"commission_amount"`
	Status              string `json:"status"`
	ConfirmAt           any    `json:"confirm_at"`
	AvailableAt         any    `json:"available_at"`
	WithdrawRequestID   uint   `json:"withdraw_request_id"`
	InvalidReason       string `json:"invalid_reason"`
	CreatedAt           any    `json:"created_at"`
	UpdatedAt           any    `json:"updated_at"`
}

// AffiliateCommissionsResponse 佣金列表
type AffiliateCommissionsResponse struct {
	Items      []AffiliateCommission `json:"items"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	Total      int64                 `json:"total"`
	TotalPages int64                 `json:"total_pages"`
}

// ListAffiliateCommissions GET /api/v1/channel/affiliate/commissions
func (c *Client) ListAffiliateCommissions(ctx context.Context, channelUserID, status string, page, pageSize int) (*AffiliateCommissionsResponse, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if status != "" {
		q.Set("status", status)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var resp PageResponse[AffiliateCommissionsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/affiliate/commissions", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// AffiliateWithdraw 提现记录
type AffiliateWithdraw struct {
	ID                 uint   `json:"id"`
	AffiliateProfileID uint   `json:"affiliate_profile_id"`
	Amount             string `json:"amount"`
	Channel            string `json:"channel"`
	Account            string `json:"account"`
	Status             string `json:"status"`
	RejectReason       string `json:"reject_reason"`
	ProcessedBy        uint   `json:"processed_by"`
	ProcessedAt        any    `json:"processed_at"`
	CreatedAt          any    `json:"created_at"`
	UpdatedAt          any    `json:"updated_at"`
}

// AffiliateWithdrawsResponse 提现列表
type AffiliateWithdrawsResponse struct {
	Items      []AffiliateWithdraw `json:"items"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	Total      int64               `json:"total"`
	TotalPages int64               `json:"total_pages"`
}

// ListAffiliateWithdraws GET /api/v1/channel/affiliate/withdraws
func (c *Client) ListAffiliateWithdraws(ctx context.Context, channelUserID, status string, page, pageSize int) (*AffiliateWithdrawsResponse, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if status != "" {
		q.Set("status", status)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var resp PageResponse[AffiliateWithdrawsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/affiliate/withdraws", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// AffiliateWithdrawRequest 提现申请
type AffiliateWithdrawRequest struct {
	IdentityPayload
	Amount  string `json:"amount" binding:"required"`
	Channel string `json:"channel" binding:"required"`
	Account string `json:"account" binding:"required"`
}

// ApplyAffiliateWithdraw POST /api/v1/channel/affiliate/withdraws
func (c *Client) ApplyAffiliateWithdraw(ctx context.Context, req AffiliateWithdrawRequest) (*AffiliateWithdraw, error) {
	var resp PageResponse[AffiliateWithdraw]
	if err := c.do(ctx, "POST", "/api/v1/channel/affiliate/withdraws", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
