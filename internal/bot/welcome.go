package bot

import (
	"fmt"
	"strings"

	"telegram-bot/internal/api"

	tele "gopkg.in/telebot.v3"
)

// onStart /start 处理：自动 provision/resolve 身份，发送欢迎消息 + 菜单
func (b *Bot) onStart(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)

	cfg := b.CurrentBotConfig()
	if cfg != nil && !cfg.Enabled {
		return c.Send(b.bundle.T(locale, "config_disabled"))
	}

	ident := buildIdentityPayload(c)
	if _, err := b.api.ProvisionTelegramIdentity(ctxFromTele(c), ident); err != nil {
		b.log.Warnw("provision identity failed", "telegram_user_id", ident.ChannelUserID, "error", err)
	}

	displayName := displayNameFromUser(c.Sender())
	message := pickLocalized(b.configWelcomeMessage(cfg), locale, b.cfg.Bot.DefaultLocale)
	if strings.TrimSpace(message) == "" {
		if cfg != nil && cfg.Welcome.Enabled {
			message = pickLocalized(cfg.Welcome.Message, locale, b.cfg.Bot.DefaultLocale)
		}
	}
	var text string
	if strings.TrimSpace(message) == "" {
		text = b.bundle.MustTr(locale, "start.welcome_no_message", map[string]any{"DisplayName": displayName})
	} else {
		text = b.bundle.MustTr(locale, "start.welcome_with_message", map[string]any{
			"DisplayName": displayName,
			"Message":     message,
		})
	}
	return b.sendMenu(c, text)
}

func (b *Bot) configWelcomeMessage(cfg *api.BotConfig) api.LocalizedText {
	if cfg == nil {
		return nil
	}
	return cfg.Welcome.Message
}

func displayNameFromUser(u *tele.User) string {
	if u == nil {
		return ""
	}
	first := strings.TrimSpace(u.FirstName)
	last := strings.TrimSpace(u.LastName)
	full := strings.TrimSpace(first + " " + last)
	if full != "" {
		return full
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return fmt.Sprintf("telegram_%d", u.ID)
}

// buildIdentityPayload 从 TG user 构造后端身份 payload
func buildIdentityPayload(c tele.Context) api.IdentityPayload {
	u := c.Sender()
	tgID := ""
	firstName := ""
	lastName := ""
	username := ""
	locale := ""
	if u != nil {
		tgID = fmt.Sprintf("%d", u.ID)
		firstName = u.FirstName
		lastName = u.LastName
		username = u.Username
		locale = u.LanguageCode
	}
	return api.IdentityPayload{
		ChannelUserID:  tgID,
		TelegramUserID: tgID,
		Username:       username,
		TelegramUser:   username,
		FirstName:      firstName,
		LastName:       lastName,
		Locale:         locale,
	}
}
