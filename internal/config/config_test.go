package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChannelFromYAMLAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
api:
  base_url: "https://shop.example.com"
channel:
  key: "yaml-key"
  secret: "yaml-secret"
bot:
  default_locale: "zh-CN"
log:
  level: "info"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 1. YAML only
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ChannelKey() != "yaml-key" || cfg.ChannelSecret() != "yaml-secret" {
		t.Fatalf("expected yaml credentials, got key=%q secret=%q", cfg.ChannelKey(), cfg.ChannelSecret())
	}

	// 2. Env overrides YAML
	t.Setenv("TG_CHANNEL_KEY", "env-key")
	t.Setenv("TG_CHANNEL_SECRET", "env-secret")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("load config with env: %v", err)
	}
	if cfg.ChannelKey() != "env-key" || cfg.ChannelSecret() != "env-secret" {
		t.Fatalf("expected env credentials, got key=%q secret=%q", cfg.ChannelKey(), cfg.ChannelSecret())
	}
}
