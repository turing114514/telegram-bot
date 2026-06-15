package bot

import (
	"context"
	"strconv"
	"strings"

	"telegram-bot/internal/i18n"
	"telegram-bot/internal/state"

	tele "gopkg.in/telebot.v3"
)

// registerHandlers 注册所有 Bot 命令与菜单按钮
func (b *Bot) registerHandlers() {
	if b.bot == nil {
		return
	}
	b.bot.Handle("/start", b.onStart)
	b.bot.Handle("/help", b.onHelp)
	b.bot.Handle("/language", b.onLanguage)
	b.bot.Handle("/cancel", b.onCancel)

	// 内置菜单 key 注册为命令
	for key, handler := range b.builtinHandlers() {
		b.bot.Handle("/"+key, handler)
	}

	// 文本兜底 + 内联键盘回调
	b.bot.Handle(tele.OnText, b.onText)
	b.bot.Handle(tele.OnCallback, b.onCallback)
}

func (b *Bot) builtinHandlers() map[string]tele.HandlerFunc {
	return map[string]tele.HandlerFunc{
		"shop_home":       b.onShopHome,
		"my_orders":       b.onMyOrders,
		"my_wallet":       b.onMyWallet,
		"affiliate":       b.onAffiliate,
		"gift_card":       b.onGiftCard,
		"switch_language": b.onLanguage,
		"contact_support": b.onContactSupport,
	}
}

// onText 兜底：把按钮文本翻译回 builtin key 后 dispatch
func (b *Bot) onText(c tele.Context) error {
	msg := strings.TrimSpace(c.Text())
	if msg == "" {
		return nil
	}
	if strings.HasPrefix(msg, "/") {
		return nil
	}
	key := ResolveMenuBuiltinByLabel(msg)
	if key != "" {
		if h, ok := b.builtinHandlers()[key]; ok {
			return h(c)
		}
	}
	return b.dispatchFreeformText(c, msg)
}

// dispatchFreeformText 根据 session 中的 awaiting 标志决定走哪条流
func (b *Bot) dispatchFreeformText(c tele.Context, text string) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	if sess.AwaitingQuantityProductID != 0 {
		pid := sess.AwaitingQuantityProductID
		sess.AwaitingQuantityProductID = 0
		qty, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || qty <= 0 {
			return c.Send(b.bundle.T(locale, "shop.recharge_invalid"))
		}
		sess.LastProductID = pid
		sess.LastProductQty = qty
		sess.PendingOrderItems = []state.OrderItemDraft{{ProductID: pid, Quantity: qty}}
		return b.handleConfirmOrder(c, locale)
	}
	if sess.AwaitingGiftCard {
		sess.AwaitingGiftCard = false
		return b.handleGiftCardRedeem(c, text, locale)
	}
	if sess.AwaitingCoupon {
		sess.AwaitingCoupon = false
		sess.PendingCoupon = text
		return b.handleConfirmOrder(c, locale)
	}
	if sess.WithdrawPendingAmount == "await_amount" {
		return b.handleWithdrawAmount(c, text, locale)
	}
	if sess.WithdrawPendingCh != "" {
		return b.handleWithdrawAccount(c, text, locale)
	}
	if sess.RechargePendingAmount == "" && looksLikeAmount(text) {
		// 收到金额输入，开始充值
		return b.handleRechargeAmount(c, text, locale)
	}
	return nil
}

// resolveUserLocale 解析用户语言：session 偏好 → TG user.language_code → config 默认
func (b *Bot) resolveUserLocale(c tele.Context, sess *state.Session) string {
	tgUserID := c.Sender().ID
	defaultLocale := b.cfg.Bot.DefaultLocale
	loc := b.state.GetLocale(tgUserID, defaultLocale)
	if tgLang := c.Sender().LanguageCode; tgLang != "" {
		normalized := i18n.NormalizeLocale(tgLang, defaultLocale)
		if normalized != "" {
			loc = normalized
		}
	}
	return loc
}

// helpers
func ctxFromTele(_ tele.Context) context.Context { return context.Background() }

func (b *Bot) sendMenu(c tele.Context, text string) error {
	menuMu.RLock()
	kb := menuMarkup
	menuMu.RUnlock()
	if kb == nil {
		return c.Send(text)
	}
	return c.Send(text, kb)
}

func looksLikeAmount(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	hasDigit := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
			continue
		}
		if r == '.' || r == ',' {
			continue
		}
		return false
	}
	return hasDigit
}
