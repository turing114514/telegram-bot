package bot

import (
	"fmt"
	"sort"
	"strings"

	"telegram-bot/internal/api"

	tele "gopkg.in/telebot.v3"
)

// onHelp /help 列出帮助中心
func (b *Bot) onHelp(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)

	cfg := b.CurrentBotConfig()
	if cfg == nil {
		return c.Send(b.bundle.T(locale, "common.loading"))
	}
	if !cfg.Help.Enabled {
		return b.sendMenu(c, b.bundle.T(locale, "help.empty"))
	}
	items := cfg.Help.Items
	enabled := make([]api.BotConfigHelpItem, 0, len(items))
	for _, it := range items {
		if it.Enabled {
			enabled = append(enabled, it)
		}
	}
	sort.SliceStable(enabled, func(i, j int) bool { return enabled[i].Order < enabled[j].Order })

	title := pickLocalized(cfg.Help.Title, locale, b.cfg.Bot.DefaultLocale)
	intro := pickLocalized(cfg.Help.Intro, locale, b.cfg.Bot.DefaultLocale)
	centerHint := pickLocalized(cfg.Help.CenterHint, locale, b.cfg.Bot.DefaultLocale)
	supportURL := ""
	if cfg.Basic.SupportURL != "" {
		supportURL = cfg.Basic.SupportURL
	}

	var sb strings.Builder
	sb.WriteString(title)
	if intro != "" {
		sb.WriteString("\n\n")
		sb.WriteString(intro)
	}
	if len(enabled) == 0 {
		sb.WriteString("\n\n")
		sb.WriteString(b.bundle.T(locale, "help.empty"))
	} else {
		for i, it := range enabled {
			summary := pickLocalized(it.Summary, locale, b.cfg.Bot.DefaultLocale)
			if summary == "" {
				summary = pickLocalized(it.Title, locale, b.cfg.Bot.DefaultLocale)
			}
			sb.WriteString(fmt.Sprintf("\n\n%d. %s", i+1, summary))
		}
	}
	if centerHint != "" {
		sb.WriteString("\n\n")
		sb.WriteString(centerHint)
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, it := range enabled {
		summary := pickLocalized(it.Summary, locale, b.cfg.Bot.DefaultLocale)
		if summary == "" {
			summary = it.Key
		}
		rows = append(rows, kb.Row(kb.Data(summary, "help", "help", it.Key)))
	}
	if supportURL != "" {
		rows = append(rows, kb.Row(kb.URL(b.bundle.T(locale, "help.support_btn"), supportURL)))
	}
	kb.Inline(rows...)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onHelpItem 显示单个帮助主题详情
func (b *Bot) onHelpItem(c tele.Context, key string) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	cfg := b.CurrentBotConfig()
	if cfg == nil {
		return c.Send(b.bundle.T(locale, "common.loading"))
	}
	var match *api.BotConfigHelpItem
	for i := range cfg.Help.Items {
		if cfg.Help.Items[i].Key == key {
			match = &cfg.Help.Items[i]
			break
		}
	}
	if match == nil || !match.Enabled {
		return b.onHelp(c)
	}
	title := pickLocalized(match.Title, locale, b.cfg.Bot.DefaultLocale)
	content := pickLocalized(match.Content, locale, b.cfg.Bot.DefaultLocale)
	var sb strings.Builder
	if title != "" {
		sb.WriteString(title)
		sb.WriteString("\n\n")
	}
	sb.WriteString(content)
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "help", "help", "list"))}
	if match.ShowSupportLink && cfg.Basic.SupportURL != "" {
		rows = append(rows, kb.Row(kb.URL(b.bundle.T(locale, "help.support_btn"), cfg.Basic.SupportURL)))
	}
	kb.Inline(rows...)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onContactSupport 展示帮助中心（带客服链接按钮）
// - 始终先列出帮助主题（用户能看到「怎么下单」「订单问题」等自助指南）
// - 如果客服链接配了，再在底部加一个跳转按钮
func (b *Bot) onContactSupport(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	cfg := b.CurrentBotConfig()

	// 优先复用 onHelp：列帮助主题（与 /help 一致）
	if cfg == nil {
		return b.onHelp(c)
	}

	var sb strings.Builder
	if !cfg.Help.Enabled || len(cfg.Help.Items) == 0 {
		// 没配帮助主题时显示原 fallback 文案
		sb.WriteString(b.bundle.T(locale, "help.support_prompt"))
	} else {
		// 列帮助主题
		title := pickLocalized(cfg.Help.Title, locale, b.cfg.Bot.DefaultLocale)
		intro := pickLocalized(cfg.Help.Intro, locale, b.cfg.Bot.DefaultLocale)
		enabled := []api.BotConfigHelpItem{}
		for _, it := range cfg.Help.Items {
			if it.Enabled {
				enabled = append(enabled, it)
			}
		}
		sort.SliceStable(enabled, func(i, j int) bool { return enabled[i].Order < enabled[j].Order })

		if title != "" {
			sb.WriteString(title)
		}
		if intro != "" {
			sb.WriteString("\n\n")
			sb.WriteString(intro)
		}
		sb.WriteString("\n")
		for i, it := range enabled {
			summary := pickLocalized(it.Summary, locale, b.cfg.Bot.DefaultLocale)
			if summary == "" {
				summary = pickLocalized(it.Title, locale, b.cfg.Bot.DefaultLocale)
			}
			sb.WriteString(fmt.Sprintf("\n%d. %s", i+1, summary))
		}
	}

	// 客服链接
	supportURL := ""
	if cfg != nil {
		supportURL = strings.TrimSpace(cfg.Basic.SupportURL)
	}

	kb := &tele.ReplyMarkup{}
	if supportURL != "" {
		// 配了客服链接：底部加跳转按钮 + 返回帮助列表
		kb.Inline(
			kb.Row(kb.URL(b.bundle.T(locale, "help.support_btn"), supportURL)),
			kb.Row(kb.Data(b.bundle.T(locale, "help.back_to_list"), "help", "help", "list")),
		)
	} else {
		// 没配：提示用户、回帮助列表
		sb.WriteString("\n\n")
		sb.WriteString(b.bundle.T(locale, "help.support_no_link"))
		kb.Inline(kb.Row(kb.Data(b.bundle.T(locale, "help.back_to_list"), "help", "help", "list")))
	}

	if c.Callback() != nil {
		return c.Edit(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onLanguage 切换语言
func (b *Bot) onLanguage(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, code := range []string{"zh-CN", "zh-TW", "en-US"} {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "lang."+code), "lang", "lang", code)))
	}
	kb.Inline(rows...)
	return c.Send(b.bundle.T(locale, "lang.title"),
		&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onCancel 取消当前输入流
func (b *Bot) onCancel(c tele.Context) error {
	b.state.Reset(c.Sender().ID)
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	return b.sendMenu(c, b.bundle.T(locale, "common.cancel"))
}
