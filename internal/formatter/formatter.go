// Package formatter 提供 Bot 端展示用的纯函数。
//
// 主要负责：TG HTML 安全转义、订单/钱包/推广视图渲染、库存文案本地化。
// 所有方法都是无状态的，locale 通过参数注入。
package formatter

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

// EscapeHTML TG HTML 模式转义。tg 的 parse_mode=HTML 严格只支持
// <b> <i> <u> <s> <code> <pre> <a href> 及其闭合；其它 < > & 必须转义。
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// EscapeMarkdownV2 TG MarkdownV2 模式（少数场景如回调 answer 使用）转义
func EscapeMarkdownV2(s string) string {
	r := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return r.Replace(s)
}

// StockText 库存文案（按 locale 返回）
func StockText(locale, status, count string) string {
	switch status {
	case "out_of_stock":
		return stockKey(locale, "out_of_stock")
	default:
		// in_stock
		if count == "-1" || count == "" {
			return stockKey(locale, "unlimited")
		}
		// 数量格式化为最简形式
		return stockKey(locale, "in_stock") + " (" + count + ")"
	}
}

func stockKey(locale, kind string) string {
	switch locale {
	case "zh-TW":
		switch kind {
		case "in_stock":
			return "現貨"
		case "out_of_stock":
			return "缺貨"
		case "unlimited":
			return "無限"
		}
	case "en-US":
		switch kind {
		case "in_stock":
			return "In stock"
		case "out_of_stock":
			return "Out of stock"
		case "unlimited":
			return "Unlimited"
		}
	}
	switch kind {
	case "in_stock":
		return "现货"
	case "out_of_stock":
		return "缺货"
	case "unlimited":
		return "无限"
	}
	return status(kind)
}

func status(s string) string { return s }

// OrderStatusLabel 订单状态文案
func OrderStatusLabel(locale, status string) string {
	switch status {
	case "pending_payment":
		return statusKey(locale, "pending_payment")
	case "paid":
		return statusKey(locale, "paid")
	case "delivered":
		return statusKey(locale, "delivered")
	case "completed":
		return statusKey(locale, "completed")
	case "cancelled", "canceled":
		return statusKey(locale, "cancelled")
	}
	return statusKey(locale, "unknown")
}

func statusKey(locale, key string) string {
	if locale == "zh-TW" {
		switch key {
		case "pending_payment":
			return "⏳ 待支付"
		case "paid":
			return "💳 已支付"
		case "delivered":
			return "✅ 已交付"
		case "completed":
			return "✅ 已完成"
		case "cancelled":
			return "❌ 已取消"
		}
		return "未知"
	}
	if locale == "en-US" {
		switch key {
		case "pending_payment":
			return "⏳ Pending payment"
		case "paid":
			return "💳 Paid"
		case "delivered":
			return "✅ Delivered"
		case "completed":
			return "✅ Completed"
		case "cancelled":
			return "❌ Cancelled"
		}
		return "Unknown"
	}
	switch key {
	case "pending_payment":
		return "⏳ 待支付"
	case "paid":
		return "💳 已支付"
	case "delivered":
		return "✅ 已交付"
	case "completed":
		return "✅ 已完成"
	case "cancelled":
		return "❌ 已取消"
	}
	return "未知"
}

// Truncate 按 rune 截断（避免中文字符被截半）
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// FormatAmount 把字符串金额渲染成本地化的"¥1,234.56"或"1,234.56 USD"
// 简化版：保留原始精度，仅加千分位
func FormatAmount(amount, currency string) string {
	if amount == "" {
		amount = "0"
	}
	// 分离整数与小数
	parts := strings.SplitN(amount, ".", 2)
	intPart := parts[0]
	decPart := ""
	if len(parts) == 2 {
		decPart = parts[1]
	}
	// 负数处理
	neg := ""
	if strings.HasPrefix(intPart, "-") {
		neg = "-"
		intPart = strings.TrimPrefix(intPart, "-")
	}
	// 千分位
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteRune(',')
		}
		b.WriteRune(ch)
	}
	intFormatted := b.String()
	if decPart != "" {
		return neg + intFormatted + "." + decPart
	}
	return neg + intFormatted
}

// DisplayCurrency 给用户看的币种（带符号）
func DisplayCurrency(currency string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "CNY", "":
		return "¥"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "JPY":
		return "¥"
	case "HKD":
		return "HK$"
	}
	return currency
}

// FormatAmountDisplay 便捷方法：组合币种符号与金额
func FormatAmountDisplay(amount, currency string) string {
	return DisplayCurrency(currency) + FormatAmount(amount, currency)
}

// ParseInt 安全解析整数（失败返回 0）
func ParseInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// ParseFloat 安全解析浮点（失败返回 0）
func ParseFloat(s string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n
}

// RoundString 保留 2 位小数
func RoundString(s string) string {
	f := ParseFloat(s)
	if math.IsNaN(f) {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", f)
}
