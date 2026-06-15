package api

import (
	"context"
	"net/url"
	"strconv"
)

// WalletAccount 钱包账户
type WalletAccount struct {
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
}

// GetWallet GET /api/v1/channel/wallet
func (c *Client) GetWallet(ctx context.Context, channelUserID string) (*WalletAccount, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	var resp PageResponse[WalletAccount]
	if err := c.do(ctx, "GET", "/api/v1/channel/wallet", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// WalletTransaction 钱包流水
type WalletTransaction struct {
	Type         string `json:"type"`
	Direction    string `json:"direction"`
	Amount       string `json:"amount"`
	BalanceAfter string `json:"balance_after"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"created_at"`
}

// WalletTransactionsResponse 钱包流水响应
type WalletTransactionsResponse struct {
	Items      []WalletTransaction `json:"items"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	Total      int64               `json:"total"`
	TotalPages int64               `json:"total_pages"`
}

// GetWalletTransactions GET /api/v1/channel/wallet/transactions
func (c *Client) GetWalletTransactions(ctx context.Context, channelUserID string, page, pageSize int) (*WalletTransactionsResponse, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var resp PageResponse[WalletTransactionsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/wallet/transactions", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GiftCardRedeemRequest 礼品卡兑换请求
type GiftCardRedeemRequest struct {
	IdentityPayload
	Code string `json:"code" binding:"required"`
}

// GiftCardRedeemResponse 礼品卡兑换响应（与后端 dto.NewGiftCardRedeemResp 对齐）
type GiftCardRedeemResponse struct {
	ID         uint   `json:"id"`
	Code       string `json:"code"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Balance    string `json:"balance"`
	RedeemedAt any    `json:"redeemed_at"`
}

// RedeemGiftCard POST /api/v1/channel/wallet/gift-card/redeem
func (c *Client) RedeemGiftCard(ctx context.Context, req GiftCardRedeemRequest) (*GiftCardRedeemResponse, error) {
	var resp PageResponse[GiftCardRedeemResponse]
	if err := c.do(ctx, "POST", "/api/v1/channel/wallet/gift-card/redeem", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// WalletRechargeRequest 钱包充值请求
type WalletRechargeRequest struct {
	IdentityPayload
	Amount    string `json:"amount" binding:"required"`
	ChannelID uint   `json:"channel_id" binding:"required"`
}

// WalletRechargeResponse 钱包充值响应
type WalletRechargeResponse struct {
	RechargeNo string `json:"recharge_no"`
	Payment    struct {
		ID              uint   `json:"id"`
		Amount          string `json:"amount"`
		FeeAmount       string `json:"fee_amount"`
		Currency        string `json:"currency"`
		Status          string `json:"status"`
		InteractionMode string `json:"interaction_mode"`
		PayURL          string `json:"pay_url"`
		QRCode          string `json:"qr_code"`
		ExpiresAt       any    `json:"expires_at"`
		WalletAddress   string `json:"wallet_address,omitempty"`
		ChainAmount     string `json:"chain_amount,omitempty"`
		Chain           string `json:"chain,omitempty"`
		TokenID         string `json:"token_id,omitempty"`
	} `json:"payment"`
}

// CreateWalletRecharge POST /api/v1/channel/wallet/recharge
func (c *Client) CreateWalletRecharge(ctx context.Context, req WalletRechargeRequest) (*WalletRechargeResponse, error) {
	var resp PageResponse[WalletRechargeResponse]
	if err := c.do(ctx, "POST", "/api/v1/channel/wallet/recharge", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
