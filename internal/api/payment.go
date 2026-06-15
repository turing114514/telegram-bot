package api

import (
	"context"
	"net/url"
	"strconv"
)

// PaymentChannelItem 支付渠道
type PaymentChannelItem struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	ProviderType    string `json:"provider_type"`
	ChannelType     string `json:"channel_type"`
	InteractionMode string `json:"interaction_mode"`
	FeeRate         string `json:"fee_rate"`
	FixedFee        string `json:"fixed_fee"`
	MinAmount       string `json:"min_amount"`
	MaxAmount       string `json:"max_amount"`
	Currency        string `json:"currency"`
}

// PaymentChannelsResponse 支付渠道列表
type PaymentChannelsResponse struct {
	Items              []PaymentChannelItem `json:"items"`
	WalletOnlyPayment  bool                 `json:"wallet_only_payment,omitempty"`
}

// GetPaymentChannels GET /api/v1/channel/payment-channels
func (c *Client) GetPaymentChannels(ctx context.Context, orderNo, channelUserID, contextParam string) (*PaymentChannelsResponse, error) {
	q := url.Values{}
	if orderNo != "" {
		q.Set("order_no", orderNo)
	}
	if channelUserID != "" {
		q.Set("channel_user_id", channelUserID)
	}
	if contextParam != "" {
		q.Set("context", contextParam)
	}
	var resp PageResponse[PaymentChannelsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/payment-channels", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// PaymentInfo 支付信息
type PaymentInfo struct {
	PaymentID       uint   `json:"payment_id"`
	OrderID         uint   `json:"order_id"`
	OrderNo         string `json:"order_no"`
	ChannelID       uint   `json:"channel_id"`
	Status          string `json:"status"`
	ProviderType    string `json:"provider_type"`
	ChannelType     string `json:"channel_type"`
	InteractionMode string `json:"interaction_mode"`
	Amount          string `json:"amount"`
	FeeRate         string `json:"fee_rate"`
	FeeAmount       string `json:"fee_amount"`
	Currency        string `json:"currency"`
	PayURL          string `json:"pay_url"`
	QRCode          string `json:"qr_code"`
	WalletAddress   string `json:"wallet_address,omitempty"`
	ChainAmount     string `json:"chain_amount,omitempty"`
	Chain           string `json:"chain,omitempty"`
	TokenID         string `json:"token_id,omitempty"`
	PaidAt          any    `json:"paid_at"`
	ExpiresAt       any    `json:"expires_at"`
	CallbackAt      any    `json:"callback_at"`
	CreatedAt       any    `json:"created_at"`
	UpdatedAt       any    `json:"updated_at"`
	TotalAmount     string `json:"total_amount"`
	WalletPaidAmount string `json:"wallet_paid_amount"`
	OnlinePaidAmount string `json:"online_paid_amount"`
	PaidAmount      string `json:"paid_amount"`
	ChannelName     string `json:"channel_name,omitempty"`
}

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
	IdentityPayload
	OrderID    uint `json:"order_id" binding:"required"`
	ChannelID  uint `json:"channel_id"`
	UseBalance bool `json:"use_balance"`
}

// CreatePayment POST /api/v1/channel/payments
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentInfo, error) {
	var resp PageResponse[PaymentInfo]
	if err := c.do(ctx, "POST", "/api/v1/channel/payments", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetLatestPayment GET /api/v1/channel/payments/latest?order_id=...
func (c *Client) GetLatestPayment(ctx context.Context, orderID uint, channelUserID string) (*PaymentInfo, error) {
	q := url.Values{}
	q.Set("order_id", strconv.FormatUint(uint64(orderID), 10))
	if channelUserID != "" {
		q.Set("channel_user_id", channelUserID)
	}
	var resp PageResponse[PaymentInfo]
	if err := c.do(ctx, "GET", "/api/v1/channel/payments/latest", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetPaymentDetail GET /api/v1/channel/payments/:id
func (c *Client) GetPaymentDetail(ctx context.Context, paymentID uint, channelUserID string) (*PaymentInfo, error) {
	q := url.Values{}
	if channelUserID != "" {
		q.Set("channel_user_id", channelUserID)
	}
	var resp PageResponse[PaymentInfo]
	path := "/api/v1/channel/payments/" + strconv.FormatUint(uint64(paymentID), 10)
	if err := c.do(ctx, "GET", path, q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
