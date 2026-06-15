// Package notify 提供 dujiao-next BotNotify 主动回调的 HTTP 接收服务。
//
// 后端 asynq worker 通过 HMAC 签名 POST 到：
//   - /internal/order-fulfilled
//   - /internal/order-paid
//   - /internal/wallet-recharge-succeeded
//
// 本服务在签名校验通过后拉取订单/钱包详情，用 i18n 推送给 TG 用户。
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"telegram-bot/internal/api"
	"telegram-bot/internal/config"
	"telegram-bot/internal/crypto"
	"telegram-bot/internal/i18n"
	"telegram-bot/internal/logger"

	tele "gopkg.in/telebot.v3"
)

// Server BotNotify HTTP 服务
type Server struct {
	api           *api.Client
	bot           *tele.Bot
	log           *logger.Logger
	secret        string
	channelKey    string
	bundle        *i18n.Bundle
	defaultLocale string
	srv           *http.Server
	maxBody       int64
	requestTimeout time.Duration
}

// NewServer 构造
func NewServer(cfg *config.Config, channelKey, channelSecret string, apiClient *api.Client, bot *tele.Bot, bundle *i18n.Bundle, log *logger.Logger) *Server {
	maxBody := cfg.Server.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	mux := http.NewServeMux()
	s := &Server{
		api:            apiClient,
		bot:            bot,
		log:            log,
		secret:         channelSecret,
		channelKey:     channelKey,
		bundle:         bundle,
		defaultLocale:  cfg.Bot.DefaultLocale,
		requestTimeout: cfg.APIRequestTimeout(),
		srv: &http.Server{
			Addr:              cfg.Server.Listen,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		maxBody: maxBody,
	}
	mux.HandleFunc("/internal/order-fulfilled", s.signMiddleware("POST", s.handleOrderFulfilled))
	mux.HandleFunc("/internal/order-paid", s.signMiddleware("POST", s.handleOrderPaid))
	mux.HandleFunc("/internal/wallet-recharge-succeeded", s.signMiddleware("POST", s.handleWalletRechargeSucceeded))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return s
}

// Start 启动 HTTP 服务
func (s *Server) Start() error {
	s.log.Infow("BotNotify HTTP server listening", "addr", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop 优雅停止
func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// signMiddleware 校验渠道签名，附带 channel key 校验
func (s *Server) signMiddleware(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		key := r.Header.Get(crypto.HeaderChannelKey)
		tsStr := r.Header.Get(crypto.HeaderTimestamp)
		sig := r.Header.Get(crypto.HeaderSignature)
		if key == "" || tsStr == "" || sig == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ts, err := crypto.ParseTimestamp(tsStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !crypto.IsTimestampValid(ts) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// 校验 channel key 是否匹配
		if key != s.channelKey {
			s.log.Warnw("bot notify wrong channel key", "got", key)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		path := r.URL.Path
		// 限制 body 大小，防止 DoS
		body, err := io.ReadAll(io.LimitReader(r.Body, s.maxBody+1))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if int64(len(body)) > s.maxBody {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		if !crypto.Verify(s.secret, method, path, sig, ts, body) {
			s.log.Warnw("bot notify signature verify failed", "path", path)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(r.Context()))
	}
}

// ---------- 事件处理 ----------

type orderNotifyPayload struct {
	OrderID        uint   `json:"order_id"`
	TelegramUserID string `json:"telegram_user_id"`
}

type walletRechargeNotifyPayload struct {
	RechargeNo     string `json:"recharge_no"`
	TelegramUserID string `json:"telegram_user_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
}

func (s *Server) handleOrderFulfilled(w http.ResponseWriter, r *http.Request) {
	var p orderNotifyPayload
	if err := bindJSON(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if p.TelegramUserID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	// 直接用 order_id 拉取，不走空 orderNo 的 GetOrderByNo
	order, err := s.api.GetOrder(ctx, p.OrderID, p.TelegramUserID, "")
	if err != nil {
		s.log.Warnw("fetch order for notify failed", "order_id", p.OrderID, "error", err)
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	payload, _ := order.FulfillmentResult.(string)
	if payload == "" {
		payload = order.FulfillmentInstructions
	}
	locale := s.localeForUser(p.TelegramUserID)
	text := s.bundle.MustTr(locale, "callback.order_fulfilled", map[string]any{
		"OrderNo": order.OrderNo,
		"Payload": payload,
	})
	if err := s.sendToTelegram(p.TelegramUserID, text); err != nil {
		s.log.Warnw("send to telegram failed", "telegram_user_id", p.TelegramUserID, "error", err)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleOrderPaid(w http.ResponseWriter, r *http.Request) {
	var p orderNotifyPayload
	if err := bindJSON(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if p.TelegramUserID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	order, err := s.api.GetOrder(ctx, p.OrderID, p.TelegramUserID, "")
	if err != nil {
		s.log.Warnw("fetch order for paid notify failed", "order_id", p.OrderID, "error", err)
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	locale := s.localeForUser(p.TelegramUserID)
	text := s.bundle.MustTr(locale, "callback.order_paid", map[string]any{
		"OrderNo":  order.OrderNo,
		"Amount":   order.TotalAmount,
		"Currency": order.Currency,
	})
	if err := s.sendToTelegram(p.TelegramUserID, text); err != nil {
		s.log.Warnw("send to telegram failed", "telegram_user_id", p.TelegramUserID, "error", err)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWalletRechargeSucceeded(w http.ResponseWriter, r *http.Request) {
	var p walletRechargeNotifyPayload
	if err := bindJSON(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if p.TelegramUserID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	wallet, err := s.api.GetWallet(ctx, p.TelegramUserID)
	if err != nil || wallet == nil {
		wallet = &api.WalletAccount{Balance: "0.00", Currency: p.Currency}
	}
	if wallet.Currency == "" {
		wallet.Currency = p.Currency
	}
	locale := s.localeForUser(p.TelegramUserID)
	text := s.bundle.MustTr(locale, "callback.wallet_recharge_succeeded", map[string]any{
		"RechargeNo": p.RechargeNo,
		"Amount":     p.Amount,
		"Currency":   p.Currency,
		"Balance":    wallet.Balance,
	})
	if err := s.sendToTelegram(p.TelegramUserID, text); err != nil {
		s.log.Warnw("send to telegram failed", "telegram_user_id", p.TelegramUserID, "error", err)
	}
	w.WriteHeader(http.StatusOK)
}

// localeForUser 从 TG user 的 language_code 推断 locale。
// BotNotify 回调只有 user_id，无法拿到 language_code，统一用 default locale。
// 后续可改为从 Redis 缓存的 user session 中读取。
func (s *Server) localeForUser(_ string) string {
	return s.defaultLocale
}

func (s *Server) sendToTelegram(tgUserID, text string) error {
	if s.bot == nil {
		return errors.New("telegram bot not initialized")
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(tgUserID), 10, 64)
	if err != nil {
		return err
	}
	_, err = s.bot.Send(&tele.User{ID: uid}, text, &tele.SendOptions{ParseMode: tele.ModeHTML})
	return err
}

func bindJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(body, v)
}
