// Package bot 包含 Telegram Bot 主体逻辑。
package bot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"telegram-bot/internal/api"
	"telegram-bot/internal/config"
	"telegram-bot/internal/i18n"
	"telegram-bot/internal/state"

	tele "gopkg.in/telebot.v3"
)

// Bot 主体
type Bot struct {
	cfg     *config.Config
	api     *api.Client
	bundle  *i18n.Bundle
	bot     *tele.Bot
	state   *state.Manager
	log     loggerLike
	machine string

	botConfig atomic.Pointer[api.BotConfig]
}

// loggerLike 解耦 *zap.SugaredLogger，方便测试
type loggerLike interface {
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)
}

// New 构造 Bot 实例（不启动）
func New(cfg *config.Config, apiClient *api.Client, bundle *i18n.Bundle, log loggerLike) (*Bot, error) {
	machine, err := resolveMachineCode(cfg.Bot.MachineCode)
	if err != nil {
		return nil, err
	}
	return &Bot{
		cfg:     cfg,
		api:     apiClient,
		bundle:  bundle,
		state:   state.NewManager(),
		log:     log,
		machine: machine,
	}, nil
}

// MachineCode 返回上报的机器码
func (b *Bot) MachineCode() string { return b.machine }

// BotVersion 暴露
func (b *Bot) BotVersion() string { return b.cfg.Bot.BotVersion }

// WebhookStatus 暴露
func (b *Bot) WebhookStatus() string { return b.cfg.Bot.WebhookStatus }

// API 暴露
func (b *Bot) API() *api.Client { return b.api }

// RawTelebot 暴露内部 telebot 实例（notify 包需要用到 bot.Send）
func (b *Bot) RawTelebot() *tele.Bot { return b.bot }

// Logger 暴露
func (b *Bot) Logger() loggerLike { return b.log }

// StateManager 暴露
func (b *Bot) StateManager() *state.Manager { return b.state }

// Bundle 暴露
func (b *Bot) Bundle() *i18n.Bundle { return b.bundle }

// Config 暴露
func (b *Bot) Config() *config.Config { return b.cfg }

// CurrentBotConfig 暴露当前生效配置（可能为 nil）
func (b *Bot) CurrentBotConfig() *api.BotConfig { return b.botConfig.Load() }

// Start 启动 Bot：拉首份配置 → 启动 poller → 启后台心跳
func (b *Bot) Start(ctx context.Context) error {
	if _, err := b.refreshConfig(ctx); err != nil {
		b.log.Warnw("initial bot config fetch failed", "error", err)
	}
	if b.botConfig.Load() == nil {
		return errors.New("bot: cannot start without initial bot config (channel client not configured?)")
	}
	if err := b.spawnTelebot(); err != nil {
		return err
	}
	go b.heartbeatLoop(ctx)
	go b.configReloadLoop(ctx)
	return nil
}

// Stop 平滑停止
func (b *Bot) Stop() {
	if b.bot != nil {
		b.bot.Stop()
	}
}

// refreshConfig 拉一份新配置；config_version 变化时通过 onConfigChange 回调刷新菜单
func (b *Bot) refreshConfig(ctx context.Context) (int, error) {
	cfg, version, err := b.api.GetBotConfig(ctx)
	if err != nil {
		return 0, err
	}
	old := b.botConfig.Load()
	b.botConfig.Store(cfg)
	if old == nil || old.ConfigVersion != cfg.ConfigVersion {
		b.log.Infow("bot config reloaded",
			"config_version", cfg.ConfigVersion,
			"enabled", cfg.Enabled,
			"menu_items", len(cfg.Menu.Items),
		)
		b.applyMenu(cfg)
	}
	return version, nil
}

// spawnTelebot 用拉到的 bot_token 实例化 telebot
func (b *Bot) spawnTelebot() error {
	cfg := b.botConfig.Load()
	if cfg == nil || strings.TrimSpace(cfg.BotToken) == "" {
		return errors.New("bot token not available in config")
	}
	settings := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: time.Duration(b.cfg.Bot.PollTimeoutSeconds) * time.Second},
	}
	tg, err := tele.NewBot(settings)
	if err != nil {
		return fmt.Errorf("create telebot: %w", err)
	}
	b.bot = tg
	b.registerHandlers()
	go func() {
		b.log.Infow("telebot polling started", "username", tg.Me.Username)
		tg.Start()
	}()
	return nil
}

// heartbeatLoop 心跳上报
func (b *Bot) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(b.cfg.UpdateInterval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.sendHeartbeat(ctx)
		}
	}
}

func (b *Bot) sendHeartbeat(ctx context.Context) {
	_, err := b.api.ReportHeartbeat(ctx, api.HeartbeatRequest{
		BotVersion:       b.BotVersion(),
		WebhookStatus:    b.WebhookStatus(),
		MachineCode:      b.machine,
		LicenseStatus:    b.cfg.Bot.LicenseStatus,
		LicenseExpiresAt: b.cfg.Bot.LicenseExpiresAt,
		Warnings:         nil,
	})
	if err != nil {
		b.log.Warnw("heartbeat failed", "error", err)
		return
	}
}

// configReloadLoop 定期重拉配置
func (b *Bot) configReloadLoop(ctx context.Context) {
	t := time.NewTicker(b.cfg.ReloadInterval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := b.refreshConfig(ctx); err != nil {
				b.log.Warnw("config reload failed", "error", err)
			}
		}
	}
}

// 菜单 markup 与 items 的内存缓存
var (
	menuMu     sync.RWMutex
	menuMarkup *tele.ReplyMarkup
	menuItems  []api.BotConfigMenuItem
)

// applyMenu 把 config.menu.items 存储并构造 markup
func (b *Bot) applyMenu(cfg *api.BotConfig) {
	items := make([]api.BotConfigMenuItem, 0, len(cfg.Menu.Items))
	for _, it := range cfg.Menu.Items {
		if !it.Enabled {
			continue
		}
		items = append(items, it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})

	menuMu.Lock()
	menuItems = items
	menuMarkup = buildReplyMarkup(cfg, items)
	menuMu.Unlock()
}

func buildReplyMarkup(cfg *api.BotConfig, items []api.BotConfigMenuItem) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	var rows []tele.Row
	for _, it := range items {
		label := pickLocalized(it.Label, cfg.DefaultLocale, cfg.DefaultLocale)
		if strings.TrimSpace(label) == "" {
			label = it.Key
		}
		var btn tele.Btn
		switch it.Action.Type {
		case "url":
			btn.URL = it.Action.Value
		case "web_app":
			btn.WebApp = &tele.WebApp{URL: it.Action.Value}
		default:
			// builtin / command 用 Btn.Contact=false，仅作文本按钮
		}
		btn.Text = label
		rows = append(rows, tele.Row{btn})
	}
	kb.Reply(rows...)
	return kb
}

// ResolveMenuLookup 返回 label -> builtin key 的查找表
func ResolveMenuLookup() map[string]string {
	menuMu.RLock()
	defer menuMu.RUnlock()
	out := make(map[string]string, len(menuItems))
	for _, it := range menuItems {
		out[it.Key] = it.Key
	}
	return out
}

// ResolveMenuBuiltinByLabel 通过用户点击的按钮文本反查 builtin key
func ResolveMenuBuiltinByLabel(label string) string {
	menuMu.RLock()
	defer menuMu.RUnlock()
	for _, it := range menuItems {
		for _, v := range it.Label {
			if v == label {
				return it.Key
			}
		}
	}
	return ""
}

// pickLocalized 取 LocalizedText 中对应 locale 的值
func pickLocalized(m api.LocalizedText, loc, def string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[loc]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	if v, ok := m[def]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	for _, k := range []string{"zh-CN", "en-US", "zh-TW"} {
		if v, ok := m[k]; ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	for _, v := range m {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// resolveMachineCode 缺省取 hostname
func resolveMachineCode(override string) (string, error) {
	override = strings.TrimSpace(override)
	if override != "" {
		return override, nil
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return fmt.Sprintf("tgbot-%d", time.Now().UnixNano()%1_000_000), nil
	}
	return host, nil
}
