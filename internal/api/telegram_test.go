package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBotConfigParsesNestedData(t *testing.T) {
	wantVer := 3
	wantToken := "123456:ABC-DEF"
	wantEnabled := true

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 简单签名：这里只测 JSON 解析，不校验签名
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code": 0,
			"msg":         "ok",
			"data": map[string]any{
				"config": map[string]any{
					"enabled":        true,
					"bot_token":      wantToken,
					"default_locale": "zh-CN",
					"config_version": wantVer,
					"basic":          map[string]any{},
					"welcome":        map[string]any{"enabled": false, "message": map[string]string{}},
					"help":           map[string]any{"enabled": false, "items": []any{}},
					"menu":           map[string]any{"items": []any{map[string]any{"key": "shop_home", "enabled": true, "order": 1, "label": map[string]string{"zh-CN": "购物"}, "action": map[string]string{"type": "builtin", "value": ""}}}},
				},
				"config_version": wantVer,
			},
		})
	}))
	defer ts.Close()

	c, err := NewClient(Config{
		BaseURL:        ts.URL,
		ChannelKey:     "key",
		ChannelSecret:  "secret",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	cfg, ver, err := c.GetBotConfig(context.Background())
	if err != nil {
		t.Fatalf("get bot config: %v", err)
	}
	if ver != wantVer {
		t.Fatalf("version = %d, want %d", ver, wantVer)
	}
	if cfg.Enabled != wantEnabled {
		t.Fatalf("enabled = %v, want %v", cfg.Enabled, wantEnabled)
	}
	if cfg.BotToken != wantToken {
		t.Fatalf("bot_token = %q, want %q", cfg.BotToken, wantToken)
	}
	if len(cfg.Menu.Items) != 1 {
		t.Fatalf("menu items = %d, want 1", len(cfg.Menu.Items))
	}
}
