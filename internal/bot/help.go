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
		rows = append(rows, kb.Row(kb.Data(summary, "help", it.Key)))
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
	rows := []tele.Row{kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "help", "list"))}
	if match.ShowSupportLink && cfg.Basic.SupportURL != "" {
		rows = append(rows, kb.Row(kb.URL(b.bundle.T(locale, "help.support_btn"), cfg.Basic.SupportURL)))
	}
	kb.Inline(rows...)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onContactSupport 直接展示客服入口
func (b *Bot) onContactSupport(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	cfg := b.CurrentBotConfig()
	text := b.bundle.T(locale, "help.support_prompt")
	kb := &tele.ReplyMarkup{}
	if cfg == nil || strings.TrimSpace(cfg.Basic.SupportURL) == "" {
		text += "\n\n" + b.bundle.T(locale, "help.support_no_link")
		kb.Inline(kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "help", "list")))
		return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	kb.Inline(
		kb.Row(kb.URL(b.bundle.T(locale, "help.support_btn"), cfg.Basic.SupportURL)),
		kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "help", "list")),
	)
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// onLanguage 切换语言
func (b *Bot) onLanguage(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, code := range []string{"zh-CN", "zh-TW", "en-US"} {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "lang."+code), "lang", code)))
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
