package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

// onCallback 处理内联键盘回调
func (b *Bot) onCallback(c tele.Context) error {
	data := c.Callback().Data
	b.log.Debugw("callback received", "data", data, "from", c.Sender().ID)
	if data == "" {
		c.Respond()
		return nil
	}
	// kb.Data(text, unique, data...) 用 | 分隔，第一个元素是 handler 前缀
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		b.log.Warnw("callback data too short", "data", data)
		c.Respond()
		return nil
	}
	switch parts[0] {
	case "help":
		c.Respond()
		return b.handleHelpCallback(c, strings.Join(parts[1:], "|"))
	case "lang":
		c.Respond()
		return b.handleLangCallback(c, parts[1])
	case "recharge":
		c.Respond()
		return b.handleRechargeCallback(c, strings.Join(parts[1:], ":"))
	case "withdraw":
		c.Respond()
		return b.handleWithdrawCallback(c, strings.Join(parts[1:], ":"))
	case "pay":
		c.Respond()
		return b.handlePayCallback(c, strings.Join(parts[1:], ":"))
	case "shop":
		c.Respond()
		return b.handleShopCallback(c, strings.Join(parts[1:], ":"))
	case "order":
		c.Respond()
		return b.handleOrderCallback(c, strings.Join(parts[1:], ":"))
	case "gift":
		c.Respond()
		return b.handleGiftCallback(c, strings.Join(parts[1:], ":"))
	case "affiliate":
		c.Respond()
		return b.handleAffiliateCallback(c, strings.Join(parts[1:], ":"))
	case "wallet":
		c.Respond()
		return b.handleWalletCallback(c, strings.Join(parts[1:], ":"))
	case "back":
		c.Respond()
		if parts[1] == "main" {
			return b.handleBackMain(c)
		}
	default:
		b.log.Warnw("unhandled callback prefix", "prefix", parts[0], "data", data)
		c.Respond()
	}
	return nil
}

func (b *Bot) handleHelpCallback(c tele.Context, action string) error {
	if action == "list" {
		return b.onHelp(c)
	}
	return b.onHelpItem(c, action)
}

func (b *Bot) handleLangCallback(c tele.Context, code string) error {
	switch code {
	case "zh-CN", "zh-TW", "en-US":
	default:
		return nil
	}
	b.state.SetLocale(c.Sender().ID, code)
	text := b.bundle.MustTr(code, "start.language_set", map[string]any{"Locale": b.bundle.T(code, "lang."+code)})
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func (b *Bot) handleBackMain(c tele.Context) error {
	cfg := b.botConfig.Load()
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ident := buildIdentityPayload(c)
	if _, err := b.api.ProvisionTelegramIdentity(ctxFromTele(c), ident); err != nil {
		b.log.Warnw("re-provision on back failed", "error", err)
	}
	displayName := displayNameFromUser(c.Sender())
	msg := ""
	if cfg != nil && cfg.Welcome.Enabled {
		msg = pickLocalized(cfg.Welcome.Message, locale, b.cfg.Bot.DefaultLocale)
	}
	if msg == "" {
		msg = b.bundle.MustTr(locale, "start.welcome_no_message", map[string]any{"DisplayName": displayName})
	} else {
		msg = b.bundle.MustTr(locale, "start.welcome_with_message", map[string]any{
			"DisplayName": displayName,
			"Message":     msg,
		})
	}
	return b.sendMenu(c, msg)
}
