// Package api 提供对 dujiao-next Channel API 的客户端封装。
//
// 所有请求统一加上渠道签名头（Dujiao-Next-Channel-Key / -Timestamp / -Signature），
// 与后端 internal/router/channel_auth.go 的 ChannelAPIAuthMiddleware 完全对齐。
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"telegram-bot/internal/crypto"
)

// Client dujiao-next Channel API 客户端
type Client struct {
	baseURL    string
	channelKey string
	secret     string
	httpClient *http.Client
}

// Config 客户端配置
type Config struct {
	BaseURL            string
	ChannelKey         string
	ChannelSecret      string
	TimeoutSeconds     int
	InsecureSkipVerify bool
}

// NewClient 构造客户端
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("api: base url is required")
	}
	if strings.TrimSpace(cfg.ChannelKey) == "" {
		return nil, errors.New("api: channel key is required")
	}
	if strings.TrimSpace(cfg.ChannelSecret) == "" {
		return nil, errors.New("api: channel secret is required")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 15
	}
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
		MaxIdleConnsPerHost: 16,
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		channelKey: cfg.ChannelKey,
		secret:     cfg.ChannelSecret,
		httpClient: &http.Client{Timeout: time.Duration(timeout) * time.Second, Transport: tr},
	}, nil
}

// ChannelError 后端返回的标准错误格式（与 internal/http/response/ChannelError 对齐）
type ChannelError struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	ErrorCode  string `json:"error_code"`
}

func (e *ChannelError) Error() string {
	return fmt.Sprintf("channel api error: http=%d code=%d error_code=%s msg=%s", e.HTTPStatus, e.Code, e.ErrorCode, e.Msg)
}

// Response 通用响应包装
type Response[T any] struct {
	StatusCode int    `json:"status_code"`
	Msg        string `json:"msg"`
	Data       T      `json:"data"`
}

// PageResponse 分页响应
type PageResponse[T any] struct {
	StatusCode int    `json:"status_code"`
	Msg        string `json:"msg"`
	Data       T      `json:"data"`
	Pagination *struct {
		Page      int   `json:"page"`
		PageSize  int   `json:"page_size"`
		Total     int64 `json:"total"`
		TotalPage int64 `json:"total_page"`
	} `json:"pagination"`
}

// do 执行签名 HTTP 请求，data 传入 nil 表示 GET
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	var bodyBytes []byte
	if body != nil {
		jb, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyBytes = jb
	}
	timestamp := time.Now().Unix()
	signature := crypto.Sign(c.secret, strings.ToUpper(method), path, timestamp, bodyBytes)

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(crypto.HeaderChannelKey, c.channelKey)
	req.Header.Set(crypto.HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(crypto.HeaderSignature, signature)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 尝试解析标准 ChannelError
		var e ChannelError
		_ = json.Unmarshal(respBody, &e)
		if e.ErrorCode == "" && e.Msg == "" {
			e.Msg = strings.TrimSpace(string(respBody))
		}
		e.HTTPStatus = resp.StatusCode
		return &e
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w (raw=%s)", err, truncate(string(respBody), 200))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// IsChannelError 检查是否为 ChannelError 类型
func IsChannelError(err error) (*ChannelError, bool) {
	var ce *ChannelError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}
