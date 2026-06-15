package api

import (
	"context"
	"net/url"
	"strconv"
)

// Category 分类项
type Category struct {
	ID           uint   `json:"id"`
	ParentID     uint   `json:"parent_id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Slug         string `json:"slug"`
	ProductCount int64  `json:"product_count"`
}

// CategoriesResponse 分类列表响应 data
type CategoriesResponse struct {
	Items []Category `json:"items"`
}

// GetCategories GET /api/v1/channel/catalog/categories
func (c *Client) GetCategories(ctx context.Context, locale string) ([]Category, error) {
	q := url.Values{}
	if locale != "" {
		q.Set("locale", locale)
	}
	var resp PageResponse[CategoriesResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/catalog/categories", q, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Items, nil
}

// WholesalePrice 批发价
type WholesalePrice struct {
	MinQuantity int    `json:"min_quantity"`
	UnitPrice   string `json:"unit_price"`
}

// ProductItem 商品列表项
type ProductItem struct {
	ID              uint             `json:"id"`
	Title           string           `json:"title"`
	Summary         string           `json:"summary"`
	ImageURL        string           `json:"image_url"`
	PriceFrom       string           `json:"price_from"`
	MemberPriceFrom string           `json:"member_price_from,omitempty"`
	WholesalePrices []WholesalePrice `json:"wholesale_prices,omitempty"`
	Currency        string           `json:"currency"`
	StockStatus     string           `json:"stock_status"`
	StockCount      int64            `json:"stock_count"`
	CategoryName    string           `json:"category_name"`
}

// ProductsResponse 商品列表响应
type ProductsResponse struct {
	Items     []ProductItem `json:"items"`
	Total     int64        `json:"total"`
	Page      int          `json:"page"`
	PageSize  int          `json:"page_size"`
	TotalPage int64        `json:"total_page"`
}

// GetProducts GET /api/v1/channel/catalog/products
func (c *Client) GetProducts(ctx context.Context, locale, categoryID string, page, pageSize int) (*ProductsResponse, error) {
	q := url.Values{}
	if locale != "" {
		q.Set("locale", locale)
	}
	if categoryID != "" {
		q.Set("category_id", categoryID)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var resp PageResponse[ProductsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/catalog/products", q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// SKUItem 商品详情 SKU
type SKUItem struct {
	ID          uint   `json:"id"`
	SKUCode     string `json:"sku_code"`
	SpecValues  string `json:"spec_values"`
	Price       string `json:"price"`
	MemberPrice string `json:"member_price,omitempty"`
	StockStatus string `json:"stock_status"`
	StockCount  int64  `json:"stock_count"`
}

// ProductDetail 商品详情
type ProductDetail struct {
	ID                 uint             `json:"id"`
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	ImageURL           string           `json:"image_url"`
	PriceFrom          string           `json:"price_from"`
	MemberPriceFrom    string           `json:"member_price_from,omitempty"`
	WholesalePrices    []WholesalePrice `json:"wholesale_prices,omitempty"`
	Currency           string           `json:"currency"`
	StockStatus        string           `json:"stock_status"`
	StockCount         int64            `json:"stock_count"`
	CategoryName       string           `json:"category_name"`
	FulfillmentType    string           `json:"fulfillment_type"`
	MinPurchaseQuantity int             `json:"min_purchase_quantity"`
	MaxPurchaseQuantity int             `json:"max_purchase_quantity"`
	ManualFormSchema   *ManualFormSchema `json:"manual_form_schema,omitempty"`
	PurchaseNote       string           `json:"purchase_note"`
	SKUs               []SKUItem        `json:"skus"`
}

// ManualFormSchema 人工表单 schema
type ManualFormSchema struct {
	Fields []ManualFormField `json:"fields"`
}

// ManualFormField 单个表单字段
type ManualFormField struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Label       string   `json:"label,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Regex       string   `json:"regex,omitempty"`
	Min         any      `json:"min,omitempty"`
	Max         any      `json:"max,omitempty"`
	MaxLen      any      `json:"max_len,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// GetProductDetail GET /api/v1/channel/catalog/products/:id
func (c *Client) GetProductDetail(ctx context.Context, id uint, locale, channelUserID string) (*ProductDetail, error) {
	q := url.Values{}
	if locale != "" {
		q.Set("locale", locale)
	}
	if channelUserID != "" {
		q.Set("channel_user_id", channelUserID)
	}
	var resp PageResponse[ProductDetail]
	path := "/api/v1/channel/catalog/products/" + strconv.FormatUint(uint64(id), 10)
	if err := c.do(ctx, "GET", path, q, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// MemberLevel 会员等级
type MemberLevel struct {
	ID                uint    `json:"id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	Icon              string  `json:"icon"`
	DiscountRate      float64 `json:"discount_rate"`
	RechargeThreshold float64 `json:"recharge_threshold"`
	SpendThreshold    float64 `json:"spend_threshold"`
	IsDefault         bool    `json:"is_default"`
	SortOrder         int     `json:"sort_order"`
}

// MemberLevelsResponse 会员等级列表响应
type MemberLevelsResponse struct {
	Items []MemberLevel `json:"items"`
}

// GetMemberLevels GET /api/v1/channel/member-levels
func (c *Client) GetMemberLevels(ctx context.Context, locale string) ([]MemberLevel, error) {
	q := url.Values{}
	if locale != "" {
		q.Set("locale", locale)
	}
	var resp PageResponse[MemberLevelsResponse]
	if err := c.do(ctx, "GET", "/api/v1/channel/member-levels", q, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Items, nil
}
