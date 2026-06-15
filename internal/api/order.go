package api

import (
	"context"
	"net/url"
	"strconv"
)

// OrderItemRequest 单个购买项
type OrderItemRequest struct {
	ProductID       uint   `json:"product_id"`
	SKUID           uint   `json:"sku_id"`
	Quantity        int    `json:"quantity"`
	FulfillmentType string `json:"fulfillment_type"`
}

// OrderRequest 通用订单字段（与后端 previewOrderRequest / createOrderRequest 对齐）
type OrderRequest struct {
	IdentityPayload
	Items          []OrderItemRequest    `json:"items"`
	ProductID      uint                  `json:"product_id"`
	SKUID          uint                  `json:"sku_id"`
	Quantity       int                   `json:"quantity"`
	CouponCode     string                `json:"coupon_code"`
	AffiliateCode  string                `json:"affiliate_code"`
	AffiliateKey   string                `json:"affiliate_visitor_key"`
	ManualFormData map[string]map[string]any `json:"manual_form_data,omitempty"`
}

// OrderPreviewItem 预览项
type OrderPreviewItem struct {
	ProductID          uint   `json:"product_id"`
	ProductTitle       string `json:"product_title"`
	SKUID              uint   `json:"sku_id"`
	SKUName            string `json:"sku_name"`
	Quantity           int    `json:"quantity"`
	OriginalUnitPrice  string `json:"original_unit_price"`
	UnitPrice          string `json:"unit_price"`
	OriginalTotalPrice string `json:"original_total_price"`
	Subtotal           string `json:"subtotal"`
	CouponDiscount     string `json:"coupon_discount"`
	PromotionDiscount  string `json:"promotion_discount"`
	WholesaleDiscount  string `json:"wholesale_discount"`
	FulfillmentType    string `json:"fulfillment_type"`
}

// OrderPreview 订单预览
type OrderPreview struct {
	ItemCount           int                `json:"item_count"`
	Items               []OrderPreviewItem `json:"items"`
	OriginalAmount      string             `json:"original_amount"`
	CouponDiscount      string             `json:"coupon_discount"`
	PromotionDiscount   string             `json:"promotion_discount"`
	WholesaleDiscount   string             `json:"wholesale_discount"`
	TotalAmount         string             `json:"total_amount"`
	Currency            string             `json:"currency"`
	Valid               bool               `json:"valid"`
	ValidationErrors    []string           `json:"validation_errors"`
}

// PreviewOrder POST /api/v1/channel/orders/preview
func (c *Client) PreviewOrder(ctx context.Context, req OrderRequest) (*OrderPreview, error) {
	var resp PageResponse[OrderPreview]
	if err := c.do(ctx, "POST", "/api/v1/channel/orders/preview", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateOrderResponse 下单结果
type CreateOrderResponse struct {
	OrderID           uint   `json:"order_id"`
	OrderNo           string `json:"order_no"`
	Status            string `json:"status"`
	FulfillmentType   string `json:"fulfillment_type"`
	Currency          string `json:"currency"`
	ItemCount         int    `json:"item_count"`
	OriginalAmount    string `json:"original_amount"`
	CouponDiscount    string `json:"coupon_discount"`
	PromotionDiscount string `json:"promotion_discount"`
	WholesaleDiscount string `json:"wholesale_discount"`
	TotalAmount       string `json:"total_amount"`
	WalletPaidAmount  string `json:"wallet_paid_amount"`
	OnlinePaidAmount  string `json:"online_paid_amount"`
	PaidAmount        string `json:"paid_amount"`
	RefundedAmount    string `json:"refunded_amount"`
	ExpiresAt         any    `json:"expires_at"`
	CreatedAt         any    `json:"created_at"`
}

// CreateOrder POST /api/v1/channel/orders
func (c *Client) CreateOrder(ctx context.Context, req OrderRequest) (*CreateOrderResponse, error) {
	var resp PageResponse[CreateOrderResponse]
	if err := c.do(ctx, "POST", "/api/v1/channel/orders", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// OrderListItem 订单列表项
type OrderListItem struct {
	OrderID          uint    `json:"order_id"`
	OrderNo          string  `json:"order_no"`
	Status           string  `json:"status"`
	Currency         string  `json:"currency"`
	TotalAmount      string  `json:"total_amount"`
	PaidAmount       string  `json:"paid_amount"`
	WalletPaidAmount string  `json:"wallet_paid_amount"`
	OnlinePaidAmount string  `json:"online_paid_amount"`
	ProductTitle     string  `json:"product_title"`
	ItemCount        int     `json:"item_count"`
	ExpiresAt        *string `json:"expires_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

// OrderListResponse 订单列表
type OrderListResponse struct {
	Items      []OrderListItem `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	Total      int64           `json:"total"`
	TotalPages int64           `json:"total_pages"`
}

// ListOrders GET /api/v1/channel/orders
func (c *Client) ListOrders(ctx context.Context, channelUserID, status, locale string, page, pageSize int) (*OrderListResponse, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if status != "" {
		q.Set("status", status)
	}
	if locale != "" {
		q.Set("locale", locale)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var resp PageResponse[OrderListResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/orders", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// OrderItemDetail 订单详情 item
type OrderItemDetail struct {
	ProductID          uint   `json:"product_id"`
	ProductTitle       string `json:"product_title"`
	SKUID              uint   `json:"sku_id"`
	SKUName            string `json:"sku_name"`
	Quantity           int    `json:"quantity"`
	OriginalUnitPrice  string `json:"original_unit_price"`
	UnitPrice          string `json:"unit_price"`
	OriginalTotalPrice string `json:"original_total_price"`
	Subtotal           string `json:"subtotal"`
	CouponDiscount     string `json:"coupon_discount"`
	PromotionDiscount  string `json:"promotion_discount"`
	WholesaleDiscount  string `json:"wholesale_discount"`
	FulfillmentType    string `json:"fulfillment_type"`
	Instructions       string `json:"instructions"`
}

// OrderFulfillmentDetail 订单 fulfillment
type OrderFulfillmentDetail struct {
	Status       string `json:"status"`
	Type         string `json:"type"`
	Payload      string `json:"payload"`
	DeliveredAt  any    `json:"delivered_at"`
	Instructions string `json:"instructions"`
}

// OrderChild 子订单
type OrderChild struct {
	OrderID    uint                   `json:"order_id"`
	OrderNo    string                 `json:"order_no"`
	Status     string                 `json:"status"`
	Fulfillment *OrderFulfillmentDetail `json:"fulfillment,omitempty"`
}

// OrderDetail 订单详情
type OrderDetail struct {
	OrderID              uint                `json:"order_id"`
	OrderNo              string              `json:"order_no"`
	Status               string              `json:"status"`
	FulfillmentType      string              `json:"fulfillment_type"`
	Currency             string              `json:"currency"`
	ItemCount            int                 `json:"item_count"`
	OriginalAmount       string              `json:"original_amount"`
	CouponDiscount       string              `json:"coupon_discount"`
	PromotionDiscount    string              `json:"promotion_discount"`
	WholesaleDiscount    string              `json:"wholesale_discount"`
	TotalAmount          string              `json:"total_amount"`
	WalletPaidAmount     string              `json:"wallet_paid_amount"`
	OnlinePaidAmount     string              `json:"online_paid_amount"`
	PaidAmount           string              `json:"paid_amount"`
	RefundedAmount       string              `json:"refunded_amount"`
	ExpiresAt            any                 `json:"expires_at"`
	CreatedAt            any                 `json:"created_at"`
	UpdatedAt            any                 `json:"updated_at"`
	PaidAt               any                 `json:"paid_at"`
	CancelledAt          any                 `json:"cancelled_at"`
	Items                []OrderItemDetail   `json:"items"`
	Children             []OrderChild        `json:"children"`
	FulfillmentStatus    string              `json:"fulfillment_status"`
	FulfillmentResult    any                 `json:"fulfillment_result"`
	FulfillmentDeliveredAt any               `json:"fulfillment_delivered_at"`
	FulfillmentInstructions string           `json:"fulfillment_instructions"`
}

// GetOrder GET /api/v1/channel/orders/:id
func (c *Client) GetOrder(ctx context.Context, orderID uint, channelUserID, locale string) (*OrderDetail, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if locale != "" {
		q.Set("locale", locale)
	}
	var resp PageResponse[OrderDetail]
	path := "/api/v1/channel/orders/" + strconv.FormatUint(uint64(orderID), 10)
	if err := c.do(ctx, "GET", path, q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetOrderByNo GET /api/v1/channel/orders/by-order-no/:order_no
func (c *Client) GetOrderByNo(ctx context.Context, orderNo, channelUserID, locale string) (*OrderDetail, error) {
	q := url.Values{}
	q.Set("channel_user_id", channelUserID)
	if locale != "" {
		q.Set("locale", locale)
	}
	var resp PageResponse[OrderDetail]
	if err := c.do(ctx, "GET", "/api/v1/channel/orders/by-order-no/"+orderNo, q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CancelOrderRequest 取消订单请求
type CancelOrderRequest struct {
	IdentityPayload
	Reason string `json:"reason"`
}

// CancelOrder POST /api/v1/channel/orders/:id/cancel
func (c *Client) CancelOrder(ctx context.Context, orderID uint, req CancelOrderRequest) (*CreateOrderResponse, error) {
	var resp PageResponse[CreateOrderResponse]
	path := "/api/v1/channel/orders/" + strconv.FormatUint(uint64(orderID), 10) + "/cancel"
	if err := c.do(ctx, "POST", path, nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
