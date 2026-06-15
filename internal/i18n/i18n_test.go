package i18n

import "testing"

func TestNewBundleAndLookup(t *testing.T) {
	b, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if got := b.T("zh-CN", "common.back"); got == "common.back" {
		t.Fatalf("missing key common.back in zh-CN: %s", got)
	}
	if got := b.T("en-US", "common.back"); got == "" {
		t.Fatal("en-US should fall back to default")
	}
}

func TestTrFallback(t *testing.T) {
	b, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if got := b.T("zh-TW", "common.back"); got == "" {
		t.Fatal("zh-TW fallback should return default")
	}
}

func TestNormalizeLocale(t *testing.T) {
	cases := []struct {
		in, def, want string
	}{
		{"zh-CN", "zh-CN", "zh-CN"},
		{"zh-Hans", "en-US", "zh-CN"},
		{"zh-TW", "en-US", "zh-TW"},
		{"zh-HK", "en-US", "zh-TW"},
		{"en", "zh-CN", "en-US"},
		{"en-GB", "zh-CN", "en-US"},
		{"fr-FR", "en-US", "en-US"},
		{"", "zh-TW", "zh-TW"},
		{"  ", "en-US", "en-US"},
	}
	for _, c := range cases {
		got := NormalizeLocale(c.in, c.def)
		if got != c.want {
			t.Errorf("NormalizeLocale(%q, %q) = %q, want %q", c.in, c.def, got, c.want)
		}
	}
}

func TestMustTr(t *testing.T) {
	b, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	got := b.MustTr("zh-CN", "orders.status_paid", nil)
	if got == "orders.status_paid" {
		t.Fatal("expected translated string")
	}
}
