package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"telegram-bot/internal/api"
	"telegram-bot/internal/formatter"
	"telegram-bot/internal/state"

	tele "gopkg.in/telebot.v3"
)

// =============== 工具函数 ===============

func (b *Bot) apiTimeout() time.Duration { return b.cfg.APIRequestTimeout() }

func (b *Bot) pageCfg() (product, order, wallet, affiliate int) {
	return b.cfg.Bot.ProductPageSize,
		b.cfg.Bot.OrderPageSize,
		b.cfg.Bot.WalletPageSize,
		b.cfg.Bot.AffiliatePageSize
}

func (b *Bot) defaultCurrency() string { return b.cfg.Bot.DefaultCurrency }

// =============== shop_home ===============

func (b *Bot) onShopHome(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()

	categories, err := b.api.GetCategories(ctx, locale)
	if err != nil {
		b.log.Warnw("get categories failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	if len(categories) == 0 {
		return c.Send(b.bundle.T(locale, "shop.empty"))
	}

	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{kb.Row(kb.Data(b.bundle.T(locale, "shop.all_products"), "shop", "shop", "list", "0"))}
	for _, cat := range categories {
		name := cat.Name
		if name == "" {
			name = fmt.Sprintf("#%d", cat.ID)
		}
		rows = append(rows, kb.Row(kb.Data(name, "shop", "shop", "cat", fmt.Sprintf("%d", cat.ID), "0")))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
	kb.Inline(rows...)
	return c.Send(b.bundle.T(locale, "shop.title"),
		&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) handleShopCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 4)
	if len(parts) == 0 {
		return nil
	}
	switch parts[0] {
	case "home":
		return b.onShopHome(c)
	case "list":
		return b.shopList(c, 0, 0)
	case "cat":
		var catID, page uint64
		if len(parts) > 1 {
			catID, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		if len(parts) > 2 {
			page, _ = strconv.ParseUint(parts[2], 10, 64)
		}
		return b.shopList(c, uint(catID), int(page))
	case "item":
		var pid uint64
		if len(parts) > 1 {
			pid, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		return b.shopItem(c, uint(pid))
	case "buy":
		var pid uint64
		var qty int
		if len(parts) > 1 {
			pid, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		if len(parts) > 2 {
			qty, _ = strconv.Atoi(parts[2])
		}
		if qty <= 0 {
			qty = 1
		}
		return b.shopBuy(c, uint(pid), qty)
	case "preview":
		locale := b.resolveUserLocale(c, b.state.Get(c.Sender().ID))
		return b.handleConfirmOrder(c, locale)
	case "coupon":
		sess := b.state.Get(c.Sender().ID)
		sess.AwaitingCoupon = true
		return c.Send(b.bundle.T(b.resolveUserLocale(c, sess), "shop.coupon_label"))
	}
	return nil
}

func (b *Bot) shopList(c tele.Context, categoryID uint, page int) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()

	_, orderPW, _, _ := b.pageCfg()
	var catIDStr string
	if categoryID > 0 {
		catIDStr = strconv.FormatUint(uint64(categoryID), 10)
	}
	resp, err := b.api.GetProducts(ctx, locale, catIDStr, page+1, orderPW)
	if err != nil {
		b.log.Warnw("get products failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	if len(resp.Items) == 0 {
		_ = c.Respond()
		return c.Send(b.bundle.T(locale, "shop.no_more_products"))
	}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, p := range resp.Items {
		title := formatter.Truncate(p.Title, 28)
		priceLine := formatter.FormatAmountDisplay(p.PriceFrom, p.Currency)
		if p.MemberPriceFrom != "" {
			priceLine = formatter.FormatAmountDisplay(p.MemberPriceFrom, p.Currency) + "/" + priceLine
		}
		label := fmt.Sprintf("%s - %s", title, priceLine)
		rows = append(rows, kb.Row(kb.Data(label, "shop", "shop", "item", fmt.Sprintf("%d", p.ID))))
	}
	if page > 0 {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_prev"), "shop", "shop", "cat", fmt.Sprintf("%d", categoryID), fmt.Sprintf("%d", page-1))))
	}
	if int64(page+1) < resp.TotalPage {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_next"), "shop", "shop", "cat", fmt.Sprintf("%d", categoryID), fmt.Sprintf("%d", page+1))))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "shop", "shop", "home")))
	kb.Inline(rows...)
	title := b.bundle.MustTr(locale, "shop.product_list_title", map[string]any{
		"Page":      page + 1,
		"TotalPage": resp.TotalPage,
	})
	if c.Callback() != nil {
		return c.Edit(title, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(title, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) shopItem(c tele.Context, productID uint) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()

	tgID := fmt.Sprintf("%d", c.Sender().ID)
	detail, err := b.api.GetProductDetail(ctx, productID, locale, tgID)
	if err != nil {
		b.log.Warnw("get product detail failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	stockText := formatter.StockText(locale, detail.StockStatus, strconv.FormatInt(detail.StockCount, 10))
	memberPrice := ""
	if detail.MemberPriceFrom != "" {
		memberPrice = formatter.FormatAmountDisplay(detail.MemberPriceFrom, detail.Currency)
	}
	text := b.bundle.MustTr(locale, "shop.product_detail", map[string]any{
		"Title":       formatter.EscapeHTML(detail.Title),
		"Description": formatter.EscapeHTML(detail.Description),
		"Price":       formatter.FormatAmount(detail.PriceFrom, detail.Currency),
		"MemberPrice": memberPrice,
		"Currency":    detail.Currency,
		"StockText":   stockText,
	})
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	if detail.StockStatus == "in_stock" {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "shop.confirm_order"), "shop", "shop", "buy", fmt.Sprintf("%d", detail.ID), "1")))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "shop", "shop", "home")))
	kb.Inline(rows...)
	if c.Callback() != nil {
		return c.Edit(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) shopBuy(c tele.Context, productID uint, qty int) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	sess.LastProductID = productID
	sess.LastProductQty = qty
	sess.PendingOrderItems = []state.OrderItemDraft{{
		ProductID: productID, Quantity: qty,
	}}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, n := range b.cfg.Bot.QuantityOptions {
		rows = append(rows, kb.Row(kb.Data(b.bundle.MustTr(locale, "shop.quantity_button", map[string]any{"Quantity": n}), "shop", "shop", "buy", fmt.Sprintf("%d", productID), fmt.Sprintf("%d", n), )))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "shop.coupon_apply"), "shop", "shop", "coupon")))
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "shop.coupon_skip"), "shop", "shop", "preview")))
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "shop", "shop", "item", fmt.Sprintf("%d", productID))))
	kb.Inline(rows...)
	if c.Callback() != nil {
		return c.Edit(b.bundle.T(locale, "shop.select_quantity"),
			&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(b.bundle.T(locale, "shop.select_quantity"),
		&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// handleConfirmOrder 先 preview 校验，再创建订单
func (b *Bot) handleConfirmOrder(c tele.Context, locale string) error {
	sess := b.state.Get(c.Sender().ID)
	if len(sess.PendingOrderItems) == 0 {
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	ident := buildIdentityPayload(c)
	items := make([]api.OrderItemRequest, 0, len(sess.PendingOrderItems))
	for _, it := range sess.PendingOrderItems {
		items = append(items, api.OrderItemRequest{
			ProductID: it.ProductID, SKUID: it.SKUID, Quantity: it.Quantity,
			FulfillmentType: it.FulfillmentType,
		})
	}
	preview, err := b.api.PreviewOrder(ctx, api.OrderRequest{
		IdentityPayload: ident,
		Items:           items,
		CouponCode:      sess.PendingCoupon,
	})
	if err != nil {
		b.log.Warnw("preview order failed", "error", err)
		if ce, ok := api.IsChannelError(err); ok {
			return c.Send(ce.Msg)
		}
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	// 显示校验结果
	if !preview.Valid {
		var sb strings.Builder
		for _, e := range preview.ValidationErrors {
			sb.WriteString("• ")
			sb.WriteString(e)
			sb.WriteString("\n")
		}
		return c.Send(b.bundle.T(locale, "common.error_generic") + "\n" + sb.String())
	}
	// 创建订单
	order, err := b.api.CreateOrder(ctx, api.OrderRequest{
		IdentityPayload: ident,
		Items:           items,
		CouponCode:      sess.PendingCoupon,
	})
	if err != nil {
		b.log.Warnw("create order failed", "error", err)
		if ce, ok := api.IsChannelError(err); ok {
			return c.Send(ce.Msg)
		}
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	text := b.bundle.MustTr(locale, "shop.order_created", map[string]any{
		"OrderNo":  order.OrderNo,
		"Total":    formatter.FormatAmount(order.TotalAmount, order.Currency),
		"Currency": order.Currency,
	})
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{
		kb.Row(kb.Data(b.bundle.T(locale, "orders.pay_now"), "pay", "pay", "list", fmt.Sprintf("%d", order.OrderID))),
		kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")),
	}
	kb.Inline(rows...)
	sess.PendingOrderItems = nil
	sess.PendingCoupon = ""
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// =============== my_orders ===============

func (b *Bot) onMyOrders(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	return b.ordersList(c, "", 0, locale)
}

func (b *Bot) ordersList(c tele.Context, status string, page int, locale string) error {
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	_, _, _, orderPA := b.pageCfg()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	resp, err := b.api.ListOrders(ctx, tgID, status, locale, page+1, orderPA)
	if err != nil {
		b.log.Warnw("list orders failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	if len(resp.Items) == 0 {
		return c.Send(b.bundle.T(locale, "orders.empty"))
	}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, o := range resp.Items {
		rows = append(rows, kb.Row(kb.Data(fmt.Sprintf("#%s · %s · %s",
				formatter.Truncate(o.OrderNo, 12),
				formatter.OrderStatusLabel(locale, o.Status),
				formatter.FormatAmountDisplay(o.TotalAmount, o.Currency)), "order", "order", "detail", fmt.Sprintf("%d", o.OrderID), )))
	}
	if page > 0 {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_prev"), "order", "order", "list", fmt.Sprintf("%d", page-1), status)))
	}
	if int64(page+1) < resp.TotalPages {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_next"), "order", "order", "list", fmt.Sprintf("%d", page+1), status)))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
	kb.Inline(rows...)
	title := b.bundle.MustTr(locale, "orders.title", map[string]any{
		"Page":      page + 1,
		"TotalPage": resp.TotalPages,
	})
	if c.Callback() != nil {
		return c.Edit(title, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(title, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) orderDetail(c tele.Context, orderID uint) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	detail, err := b.api.GetOrder(ctx, orderID, tgID, locale)
	if err != nil {
		b.log.Warnw("get order failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	var itemsText strings.Builder
	for _, it := range detail.Items {
		itemsText.WriteString(b.bundle.MustTr(locale, "orders.item_line", map[string]any{
			"Title":    formatter.EscapeHTML(it.ProductTitle),
			"Quantity": it.Quantity,
			"Subtotal": formatter.FormatAmount(it.Subtotal, detail.Currency),
			"Currency": detail.Currency,
		}))
		itemsText.WriteString("\n")
	}
	fulfillmentText := b.bundle.T(locale, "orders.fulfillment_pending")
	if detail.FulfillmentStatus == "delivered" || detail.FulfillmentStatus == "completed" {
		payload, _ := detail.FulfillmentResult.(string)
		if payload == "" {
			payload = detail.FulfillmentInstructions
		}
		if payload != "" {
			fulfillmentText = b.bundle.MustTr(locale, "orders.fulfillment_ready", map[string]any{
				"Payload": formatter.EscapeHTML(payload),
			})
		}
	}
	text := b.bundle.MustTr(locale, "orders.order_detail", map[string]any{
		"OrderNo":     detail.OrderNo,
		"StatusLabel": formatter.OrderStatusLabel(locale, detail.Status),
		"Total":       formatter.FormatAmount(detail.TotalAmount, detail.Currency),
		"Currency":    detail.Currency,
		"Paid":        formatter.FormatAmount(detail.PaidAmount, detail.Currency),
		"WalletPaid":  formatter.FormatAmount(detail.WalletPaidAmount, detail.Currency),
		"OnlinePaid":  formatter.FormatAmount(detail.OnlinePaidAmount, detail.Currency),
		"Items":       itemsText.String(),
		"Fulfillment": fulfillmentText,
	})
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	if detail.Status == "pending_payment" {
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "orders.pay_now"), "pay", "pay", "list", fmt.Sprintf("%d", detail.OrderID))))
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "orders.cancel_order"), "order", "order", "cancel", fmt.Sprintf("%d", detail.OrderID))))
	}
	rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "order", "order", "list", "0", "")))
	kb.Inline(rows...)
	if c.Callback() != nil {
		return c.Edit(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) handleOrderCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 3)
	switch parts[0] {
	case "list":
		var page int
		var status string
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			status = parts[2]
		}
		sess := b.state.Get(c.Sender().ID)
		locale := b.resolveUserLocale(c, sess)
		return b.ordersList(c, status, page, locale)
	case "detail":
		var id uint64
		if len(parts) > 1 {
			id, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		return b.orderDetail(c, uint(id))
	case "cancel":
		var id uint64
		if len(parts) > 1 {
			id, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		return b.cancelOrder(c, uint(id))
	}
	return nil
}

func (b *Bot) cancelOrder(c tele.Context, orderID uint) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	if _, err := b.api.CancelOrder(ctx, orderID, api.CancelOrderRequest{
		IdentityPayload: api.IdentityPayload{ChannelUserID: tgID, TelegramUserID: tgID},
	}); err != nil {
		if ce, ok := api.IsChannelError(err); ok {
			return c.Send(ce.Msg)
		}
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	if c.Callback() != nil {
		_ = c.Respond()
	}
	return c.Send(b.bundle.T(locale, "orders.cancelled"))
}

// =============== my_wallet ===============

func (b *Bot) onMyWallet(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	wallet, err := b.api.GetWallet(ctx, tgID)
	if err != nil {
		b.log.Warnw("get wallet failed", "error", err)
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	currency := wallet.Currency
	if currency == "" {
		currency = b.defaultCurrency()
	}
	text := b.bundle.MustTr(locale, "wallet.title_with_balance", map[string]any{
		"Label":    b.bundle.T(locale, "wallet.balance"),
		"Balance":  formatter.FormatAmount(wallet.Balance, currency),
		"Currency": currency,
	})
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{
		kb.Row(kb.Data(b.bundle.T(locale, "wallet.recharge"), "recharge", "recharge", "start")),
		kb.Row(kb.Data(b.bundle.T(locale, "wallet.transactions"), "wallet", "wallet", "txn", "0")),
		kb.Row(kb.Data(b.bundle.T(locale, "menu.gift_card"), "gift", "gift", "start")),
		kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")),
	}
	kb.Inline(rows...)
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) handleRechargeCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 3)
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	switch parts[0] {
	case "start":
		sess.RechargePendingAmount = ""
		return c.Send(b.bundle.T(locale, "wallet.recharge_amount_prompt"))
	case "amount":
		if len(parts) < 2 {
			return nil
		}
		return b.handleRechargeAmount(c, parts[1], locale)
	case "do":
		if len(parts) < 2 {
			return nil
		}
		inner := strings.SplitN(parts[1], "|", 2)
		if len(inner) < 2 {
			return nil
		}
		amount := inner[0]
		channelID, _ := strconv.ParseUint(inner[1], 10, 64)
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		ident := buildIdentityPayload(c)
		resp, err := b.api.CreateWalletRecharge(ctx, api.WalletRechargeRequest{
			IdentityPayload: ident,
			Amount:          amount,
			ChannelID:       uint(channelID),
		})
		if err != nil {
			if ce, ok := api.IsChannelError(err); ok {
				return c.Send(ce.Msg)
			}
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		sess.RechargePendingAmount = ""
		text := b.bundle.MustTr(locale, "wallet.recharge_created", map[string]any{
			"Amount":   formatter.FormatAmount(resp.Payment.Amount, resp.Payment.Currency),
			"Currency": resp.Payment.Currency,
		})
		if resp.Payment.PayURL != "" {
			text += "\n" + b.bundle.T(locale, "orders.pay_url_label") + ": " + resp.Payment.PayURL
		}
		if resp.Payment.QRCode != "" {
			text += "\n" + b.bundle.T(locale, "orders.qr_label") + ": " + resp.Payment.QRCode
		}
		return c.Send(text)
	}
	return nil
}

func (b *Bot) handleRechargeAmount(c tele.Context, amount, locale string) error {
	amt, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil || amt <= 0 {
		return c.Send(b.bundle.T(locale, "wallet.recharge_invalid"))
	}
	sess := b.state.Get(c.Sender().ID)
	sess.RechargePendingAmount = strings.TrimSpace(amount)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	ch, err := b.api.GetPaymentChannels(ctx, "", tgID, "recharge")
	if err != nil {
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	eligible := make([]api.PaymentChannelItem, 0, len(ch.Items))
	for _, item := range ch.Items {
		if !isAmountInRange(amt, item.MinAmount, item.MaxAmount) {
			continue
		}
		eligible = append(eligible, item)
	}
	if len(eligible) == 0 {
		var minS, maxS string
		if len(ch.Items) > 0 {
			minS = ch.Items[0].MinAmount
			maxS = ch.Items[0].MaxAmount
		}
		if minS == "0" {
			minS = ""
		}
		if maxS == "0" {
			maxS = ""
		}
		text := b.bundle.T(locale, "wallet.recharge_invalid")
		if minS != "" || maxS != "" {
			hint := b.bundle.MustTr(locale, "wallet.recharge_range_hint", map[string]any{
				"Min": minS,
				"Max": maxS,
			})
			text += " " + hint
		}
		return c.Send(text)
	}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, item := range eligible {
		rows = append(rows, kb.Row(kb.Data(item.Name, "recharge", "recharge", "do", amount+"|"+fmt.Sprintf("%d", item.ID))))
	}
	kb.Inline(rows...)
	return c.Send(b.bundle.T(locale, "wallet.recharge_channel_prompt"),
		&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func isAmountInRange(amount float64, minS, maxS string) bool {
	if minS != "" && minS != "0" {
		if mn, err := strconv.ParseFloat(minS, 64); err == nil && amount < mn {
			return false
		}
	}
	if maxS != "" && maxS != "0" {
		if mx, err := strconv.ParseFloat(maxS, 64); err == nil && amount > mx {
			return false
		}
	}
	return true
}

func (b *Bot) handleWalletCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 2)
	if len(parts) < 1 {
		return nil
	}
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	switch parts[0] {
	case "txn":
		var page int
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		_, _, walletPW, _ := b.pageCfg()
		tgID := fmt.Sprintf("%d", c.Sender().ID)
		wallet, wErr := b.api.GetWallet(ctx, tgID)
		currency := b.defaultCurrency()
		if wErr == nil && wallet != nil && wallet.Currency != "" {
			currency = wallet.Currency
		}
		resp, err := b.api.GetWalletTransactions(ctx, tgID, page+1, walletPW)
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		if len(resp.Items) == 0 {
			return c.Send(b.bundle.T(locale, "wallet.empty_txn"))
		}
		var sb strings.Builder
		for _, t := range resp.Items {
			sb.WriteString(b.bundle.MustTr(locale, "wallet.txn_line", map[string]any{
				"Time":           t.CreatedAt,
				"TypeLabel":      b.bundle.MustTr(locale, "wallet.type_"+t.Type, nil),
				"DirectionLabel": b.bundle.MustTr(locale, "wallet.direction_"+t.Direction, nil),
				"Amount":         formatter.FormatAmount(t.Amount, currency),
				"Currency":       currency,
				"BalanceAfter":   formatter.FormatAmount(t.BalanceAfter, currency),
			}))
			sb.WriteString("\n")
		}
		kb := &tele.ReplyMarkup{}
		rows := []tele.Row{}
		if page > 0 {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_prev"), "wallet", "wallet", "txn", fmt.Sprintf("%d", page-1))))
		}
		if int64(page+1) < resp.TotalPages {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_next"), "wallet", "wallet", "txn", fmt.Sprintf("%d", page+1))))
		}
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
		kb.Inline(rows...)
		if c.Callback() != nil {
			return c.Edit(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
		}
		return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	return nil
}

// =============== gift_card ===============

func (b *Bot) onGiftCard(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	sess.AwaitingGiftCard = true
	return c.Send(b.bundle.T(locale, "gift_card.prompt"))
}

func (b *Bot) handleGiftCardRedeem(c tele.Context, code, locale string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return c.Send(b.bundle.T(locale, "gift_card.invalid"))
	}
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	ident := buildIdentityPayload(c)
	resp, err := b.api.RedeemGiftCard(ctx, api.GiftCardRedeemRequest{
		IdentityPayload: ident,
		Code:            code,
	})
	if err != nil {
		if ce, ok := api.IsChannelError(err); ok {
			switch ce.ErrorCode {
			case "gift_card_not_found":
				return c.Send(b.bundle.T(locale, "gift_card.not_found"))
			case "gift_card_expired":
				return c.Send(b.bundle.T(locale, "gift_card.expired"))
			case "gift_card_disabled":
				return c.Send(b.bundle.T(locale, "gift_card.disabled"))
			case "gift_card_redeemed":
				return c.Send(b.bundle.T(locale, "gift_card.redeemed"))
			}
			return c.Send(ce.Msg)
		}
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	currency := resp.Currency
	if currency == "" {
		currency = b.defaultCurrency()
	}
	text := b.bundle.MustTr(locale, "gift_card.success", map[string]any{
		"Amount":   formatter.FormatAmount(resp.Amount, currency),
		"Currency": currency,
		"Balance":  formatter.FormatAmount(resp.Balance, currency),
	})
	return c.Send(text)
}

func (b *Bot) handleGiftCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 2)
	if parts[0] == "start" {
		return b.onGiftCard(c)
	}
	return nil
}

// =============== affiliate ===============

func (b *Bot) onAffiliate(c tele.Context) error {
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	dash, err := b.api.GetAffiliateDashboard(ctx, tgID)
	if err != nil {
		ce, _ := api.IsChannelError(err)
		if ce == nil || ce.HTTPStatus >= 500 {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		dash = &api.AffiliateDashboard{}
	}
	if !dash.Opened {
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data(b.bundle.T(locale, "affiliate.open"), "affiliate", "affiliate", "open")),
			kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")),
		)
		return c.Send(b.bundle.T(locale, "affiliate.not_opened"),
			&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	}
	conv := dash.ConversionRate
	if conv == "" {
		conv = "0%"
	}
	currency := b.defaultCurrency()
	if wallet, werr := b.api.GetWallet(ctx, tgID); werr == nil && wallet != nil && wallet.Currency != "" {
		currency = wallet.Currency
	}
	text := b.bundle.MustTr(locale, "affiliate.stats_title", map[string]any{
		"Click":      dash.ClickCount,
		"Orders":     dash.ValidOrderCount,
		"Conversion": conv,
		"Pending":    formatter.FormatAmount(dash.PendingCommission, currency),
		"Available":  formatter.FormatAmount(dash.AvailableCommission, currency),
		"Withdrawn":  formatter.FormatAmount(dash.WithdrawnCommission, currency),
	})
	if dash.AffiliateCode != "" {
		text += "\n\n" + b.bundle.T(locale, "affiliate.code_label") + ": `" + dash.AffiliateCode + "`"
	}
	if dash.PromotionPath != "" {
		text += "\n" + b.bundle.T(locale, "affiliate.promotion_path_label") + ": " + dash.PromotionPath
	}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{
		kb.Row(kb.Data(b.bundle.T(locale, "affiliate.commissions"), "affiliate", "affiliate", "commissions", "0")),
		kb.Row(kb.Data(b.bundle.T(locale, "affiliate.withdraws"), "affiliate", "affiliate", "withdraws", "0")),
		kb.Row(kb.Data(b.bundle.T(locale, "affiliate.withdraw_apply"), "affiliate", "affiliate", "withdraw", "start")),
		kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")),
	}
	kb.Inline(rows...)
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

func (b *Bot) handleAffiliateCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 3)
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	switch parts[0] {
	case "open":
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		ident := buildIdentityPayload(c)
		profile, err := b.api.OpenAffiliateAndGet(ctx, ident)
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		text := b.bundle.MustTr(locale, "affiliate.open_success", map[string]any{"Code": profile.Code})
		return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML})
	case "commissions":
		var page int
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		_, _, _, affPA := b.pageCfg()
		tgID := fmt.Sprintf("%d", c.Sender().ID)
		currency := b.defaultCurrency()
		if wallet, werr := b.api.GetWallet(ctx, tgID); werr == nil && wallet != nil && wallet.Currency != "" {
			currency = wallet.Currency
		}
		resp, err := b.api.ListAffiliateCommissions(ctx, tgID, "", page+1, affPA)
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		if len(resp.Items) == 0 {
			return c.Send(b.bundle.T(locale, "affiliate.commissions_empty"))
		}
		var sb strings.Builder
		for _, cm := range resp.Items {
			sb.WriteString(fmt.Sprintf("• #%s · %s · %s %s\n",
				cm.OrderNo, cm.Status,
				formatter.FormatAmount(cm.CommissionAmount, currency), currency))
		}
		kb := &tele.ReplyMarkup{}
		rows := []tele.Row{}
		if page > 0 {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_prev"), "affiliate", "affiliate", "commissions", fmt.Sprintf("%d", page-1))))
		}
		if int64(page+1) < resp.TotalPages {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_next"), "affiliate", "affiliate", "commissions", fmt.Sprintf("%d", page+1))))
		}
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
		kb.Inline(rows...)
		if c.Callback() != nil {
			return c.Edit(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
		}
		return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	case "withdraws":
		var page int
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		_, _, _, affPA := b.pageCfg()
		tgID := fmt.Sprintf("%d", c.Sender().ID)
		currency := b.defaultCurrency()
		if wallet, werr := b.api.GetWallet(ctx, tgID); werr == nil && wallet != nil && wallet.Currency != "" {
			currency = wallet.Currency
		}
		resp, err := b.api.ListAffiliateWithdraws(ctx, tgID, "", page+1, affPA)
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		if len(resp.Items) == 0 {
			return c.Send(b.bundle.T(locale, "affiliate.withdraws_empty"))
		}
		var sb strings.Builder
		for _, w := range resp.Items {
			sb.WriteString(fmt.Sprintf("• %s · %s · %s %s\n",
				w.Status, w.Channel, formatter.FormatAmount(w.Amount, currency), currency))
		}
		kb := &tele.ReplyMarkup{}
		rows := []tele.Row{}
		if page > 0 {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_prev"), "affiliate", "affiliate", "withdraws", fmt.Sprintf("%d", page-1))))
		}
		if int64(page+1) < resp.TotalPages {
			rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.page_next"), "affiliate", "affiliate", "withdraws", fmt.Sprintf("%d", page+1))))
		}
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
		kb.Inline(rows...)
		if c.Callback() != nil {
			return c.Edit(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
		}
		return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	case "withdraw":
		if len(parts) > 1 && parts[1] == "start" {
			ctx2, cancel2 := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
			defer cancel2()
			tgID2 := fmt.Sprintf("%d", c.Sender().ID)
			dash, err := b.api.GetAffiliateDashboard(ctx2, tgID2)
			min := "0"
			if err == nil && dash != nil {
				min = dash.MinWithdrawAmount
			}
			text := b.bundle.MustTr(locale, "affiliate.withdraw_amount_prompt", map[string]any{"Min": min})
			sess.WithdrawPendingAmount = "await_amount"
			return c.Send(text)
		}
	}
	return nil
}

func (b *Bot) handleWithdrawCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 2)
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	switch parts[0] {
	case "channel":
		if len(parts) < 2 {
			return nil
		}
		sess.WithdrawPendingCh = parts[1]
		return c.Send(b.bundle.T(locale, "affiliate.withdraw_account_prompt"))
	}
	return nil
}

// handleWithdrawAmount 处理用户输入的提现金额
func (b *Bot) handleWithdrawAmount(c tele.Context, amount, locale string) error {
	sess := b.state.Get(c.Sender().ID)
	if sess.WithdrawPendingAmount != "await_amount" {
		return nil
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(amount), 64); err != nil {
		return c.Send(b.bundle.T(locale, "wallet.recharge_invalid"))
	}
	sess.WithdrawPendingAmount = strings.TrimSpace(amount)
	// 从后端 affiliate dashboard 拉取可用提现渠道
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	tgID := fmt.Sprintf("%d", c.Sender().ID)
	dash, err := b.api.GetAffiliateDashboard(ctx, tgID)
	channels := []string{}
	if err == nil && dash != nil && len(dash.WithdrawChannels) > 0 {
		for _, ch := range dash.WithdrawChannels {
			switch v := ch.(type) {
			case string:
				if s := strings.TrimSpace(v); s != "" {
					channels = append(channels, s)
				}
			case map[string]any:
				if s, ok := v["key"].(string); ok && s != "" {
					channels = append(channels, s)
				} else if s, ok := v["name"].(string); ok && s != "" {
					channels = append(channels, s)
				}
			}
		}
	}
	if len(channels) == 0 {
		channels = []string{"alipay", "wechat", "usdt_trc20"}
	}
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	for _, ch := range channels {
		rows = append(rows, kb.Row(kb.Data(ch, "withdraw", "withdraw", "channel", ch)))
	}
	kb.Inline(rows...)
	return c.Send(b.bundle.T(locale, "affiliate.withdraw_channel_prompt"),
		&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
}

// handleWithdrawAccount 处理用户输入的提现账号
func (b *Bot) handleWithdrawAccount(c tele.Context, account, locale string) error {
	sess := b.state.Get(c.Sender().ID)
	// 状态校验：必须在 amount→channel→account 的正确流程中
	if sess.WithdrawPendingAmount == "" || sess.WithdrawPendingAmount == "await_amount" || sess.WithdrawPendingCh == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
	defer cancel()
	ident := buildIdentityPayload(c)
	_, err := b.api.ApplyAffiliateWithdraw(ctx, api.AffiliateWithdrawRequest{
		IdentityPayload: ident,
		Amount:          sess.WithdrawPendingAmount,
		Channel:         sess.WithdrawPendingCh,
		Account:         account,
	})
	// 先记金额，再重置
	amount := sess.WithdrawPendingAmount
	sess.WithdrawPendingAmount = ""
	sess.WithdrawPendingCh = ""
	if err != nil {
		if ce, ok := api.IsChannelError(err); ok {
			return c.Send(ce.Msg)
		}
		return c.Send(b.bundle.T(locale, "common.error_generic"))
	}
	currency := b.defaultCurrency()
	if wallet, werr := b.api.GetWallet(ctxFromTele(c), fmt.Sprintf("%d", c.Sender().ID)); werr == nil && wallet != nil && wallet.Currency != "" {
		currency = wallet.Currency
	}
	text := b.bundle.MustTr(locale, "affiliate.withdraw_success", map[string]any{
		"Amount":   formatter.FormatAmount(amount, currency),
		"Currency": currency,
	})
	return c.Send(text)
}

// =============== payment ===============

func (b *Bot) handlePayCallback(c tele.Context, action string) error {
	parts := strings.SplitN(action, ":", 3)
	sess := b.state.Get(c.Sender().ID)
	locale := b.resolveUserLocale(c, sess)
	switch parts[0] {
	case "list":
		var orderID uint64
		if len(parts) > 1 {
			orderID, _ = strconv.ParseUint(parts[1], 10, 64)
		}
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		tgID := fmt.Sprintf("%d", c.Sender().ID)
		order, err := b.api.GetOrder(ctx, uint(orderID), tgID, locale)
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		ch, err := b.api.GetPaymentChannels(ctx, order.OrderNo, tgID, "")
		if err != nil {
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		kb := &tele.ReplyMarkup{}
		rows := []tele.Row{}
		for _, ch := range ch.Items {
			rows = append(rows, kb.Row(kb.Data(ch.Name, "pay", "pay", "do", fmt.Sprintf("%d|%d", orderID, ch.ID))))
		}
		rows = append(rows, kb.Row(kb.Data(b.bundle.T(locale, "common.back"), "back", "back", "main")))
		kb.Inline(rows...)
		return c.Send(b.bundle.T(locale, "orders.pay_title"),
			&tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: kb})
	case "do":
		if len(parts) < 2 {
			return nil
		}
		inner := strings.SplitN(parts[1], "|", 2)
		if len(inner) < 2 {
			return nil
		}
		orderID, _ := strconv.ParseUint(inner[0], 10, 64)
		channelID, _ := strconv.ParseUint(inner[1], 10, 64)
		ctx, cancel := context.WithTimeout(ctxFromTele(c), b.apiTimeout())
		defer cancel()
		ident := buildIdentityPayload(c)
		resp, err := b.api.CreatePayment(ctx, api.CreatePaymentRequest{
			IdentityPayload: ident,
			OrderID:         uint(orderID),
			ChannelID:       uint(channelID),
		})
		if err != nil {
			if ce, ok := api.IsChannelError(err); ok {
				return c.Send(ce.Msg)
			}
			return c.Send(b.bundle.T(locale, "common.error_generic"))
		}
		var text strings.Builder
		text.WriteString(b.bundle.MustTr(locale, "orders.pay_created", map[string]any{
			"Amount":   formatter.FormatAmount(resp.Amount, resp.Currency),
			"Currency": resp.Currency,
		}))
		if resp.ChannelName != "" {
			text.WriteString("\n")
			text.WriteString(b.bundle.MustTr(locale, "orders.pay_choose_channel", map[string]any{"Name": resp.ChannelName}))
		}
		if resp.PayURL != "" {
			text.WriteString("\n")
			text.WriteString(b.bundle.T(locale, "orders.pay_url_label") + ": " + resp.PayURL)
		}
		if resp.QRCode != "" {
			text.WriteString("\n")
			text.WriteString(b.bundle.T(locale, "orders.qr_label") + ": " + resp.QRCode)
		}
		return c.Send(text.String())
	}
	return nil
}
