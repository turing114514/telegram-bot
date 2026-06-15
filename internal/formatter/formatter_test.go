package formatter

import "testing"

func TestStockText(t *testing.T) {
	if got := StockText("zh-CN", "in_stock", "5"); got == "" {
		t.Fatal("empty stock text")
	}
	if got := StockText("en-US", "out_of_stock", "0"); got != "Out of stock" {
		t.Fatalf("en-US out_of_stock wrong: %s", got)
	}
	if got := StockText("zh-CN", "in_stock", "-1"); got == "" {
		t.Fatal("zh-CN unlimited empty")
	}
}

func TestOrderStatusLabel(t *testing.T) {
	cases := []struct {
		locale, status, want string
	}{
		{"zh-CN", "pending_payment", "⏳ 待支付"},
		{"zh-TW", "paid", "💳 已支付"},
		{"en-US", "delivered", "✅ Delivered"},
		{"zh-CN", "unknown_xyz", "未知"},
	}
	for _, c := range cases {
		if got := OrderStatusLabel(c.locale, c.status); got != c.want {
			t.Errorf("OrderStatusLabel(%q,%q) = %q, want %q", c.locale, c.status, got, c.want)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	if got := FormatAmount("1234.56", "CNY"); got != "1,234.56" {
		t.Fatalf("amount: %s", got)
	}
	if got := FormatAmount("1000000", ""); got != "1,000,000" {
		t.Fatalf("big amount: %s", got)
	}
	if got := FormatAmountDisplay("99.00", "USD"); got != "$99.00" {
		t.Fatalf("display usd: %s", got)
	}
}

func TestEscapeHTML(t *testing.T) {
	if got := EscapeHTML("<b>hi</b>"); got != "&lt;b&gt;hi&lt;/b&gt;" {
		t.Fatalf("escape: %s", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("中文abcde", 3); got != "中文a…" {
		t.Fatalf("truncate: %s", got)
	}
	// 长度未超出时直接返回
	if got := Truncate("中文", 5); got != "中文" {
		t.Fatalf("truncate short: %s", got)
	}
}
