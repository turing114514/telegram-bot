// Package i18n 提供 Bot 端的多语言文案加载与查询。
//
// locale JSON 通过 //go:embed 嵌入二进制，支持 zh-CN / zh-TW / en-US。
// 模板占位符采用 Go text/template 语法（与 dujiao-next 一致）。
package i18n

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

//go:embed locales/*.json
var localeFS embed.FS

// SupportedLocales 与 dujiao-next constants.SupportedLocales 对齐
var SupportedLocales = []string{"zh-CN", "zh-TW", "en-US"}

// IsSupportedLocale 判断 locale 是否受支持
func IsSupportedLocale(locale string) bool {
	for _, l := range SupportedLocales {
		if l == locale {
			return true
		}
	}
	return false
}

// Bundle 多语言资源
type Bundle struct {
	mu       sync.RWMutex
	entries  map[string]map[string]string // locale -> key -> template
	defaults map[string]string
}

// NewBundle 从嵌入资源构建多语言 bundle
func NewBundle(defaultLocale string) (*Bundle, error) {
	defaultLocale = strings.TrimSpace(defaultLocale)
	if !IsSupportedLocale(defaultLocale) {
		defaultLocale = "zh-CN"
	}
	b := &Bundle{
		entries:  make(map[string]map[string]string),
		defaults: make(map[string]string),
	}
	for _, loc := range SupportedLocales {
		raw, err := localeFS.ReadFile("locales/" + loc + ".json")
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", loc, err)
		}
		var tree map[string]any
		if err := json.Unmarshal(raw, &tree); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", loc, err)
		}
		flat := flattenTree(tree, "")
		b.entries[loc] = flat
	}
	// 默认值：优先用 defaultLocale
	if def, ok := b.entries[defaultLocale]; ok {
		b.defaults = def
	} else if def, ok := b.entries["zh-CN"]; ok {
		b.defaults = def
	}
	return b, nil
}

// T 取纯字符串
func (b *Bundle) T(locale, key string) string {
	v, ok := b.lookup(locale, key)
	if !ok {
		if def, ok := b.defaults[key]; ok {
			return def
		}
		return key
	}
	return v
}

// Tr 取带占位符的模板字符串并渲染
func (b *Bundle) Tr(locale, key string, data map[string]any) (string, error) {
	tplStr, ok := b.lookup(locale, key)
	if !ok {
		if def, ok := b.defaults[key]; ok {
			tplStr = def
		} else {
			tplStr = key
		}
	}
	tpl, err := template.New("i18n").Option("missingkey=zero").Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("parse template %s.%s: %w", locale, key, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s.%s (%q): %w", locale, key, tplStr, err)
	}
	return buf.String(), nil
}

// MustTr 模板渲染失败时降级为纯字符串
func (b *Bundle) MustTr(locale, key string, data map[string]any) string {
	s, err := b.Tr(locale, key, data)
	if err != nil {
		return b.T(locale, key)
	}
	return s
}

func (b *Bundle) lookup(locale, key string) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if m, ok := b.entries[locale]; ok {
		if v, ok := m[key]; ok {
			return v, true
		}
	}
	// fallback 到默认 locale
	if def, ok := b.defaults[key]; ok {
		return def, true
	}
	return "", false
}

func flattenTree(tree map[string]any, prefix string) map[string]string {
	out := make(map[string]string)
	for k, v := range tree {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch typed := v.(type) {
		case string:
			out[key] = typed
		case map[string]any:
			for k2, v2 := range flattenTree(typed, key) {
				out[k2] = v2
			}
		default:
			// 忽略非字符串节点
		}
	}
	return out
}

// NormalizeLocale 归一化 locale（支持 IETF tag 简化）
func NormalizeLocale(locale, fallback string) string {
	l := strings.TrimSpace(locale)
	if l == "" {
		l = fallback
	}
	// 取 - 前的语言段，匹配最接近的 SupportedLocales
	primary := strings.SplitN(l, "-", 2)[0]
	switch strings.ToLower(primary) {
	case "zh":
		// 区分简体/繁体：用 full tag 二次判断
		if strings.EqualFold(l, "zh-tw") || strings.EqualFold(l, "zh-Hant") || strings.EqualFold(l, "zh-HK") {
			return "zh-TW"
		}
		return "zh-CN"
	case "en":
		return "en-US"
	}
	if IsSupportedLocale(l) {
		return l
	}
	if IsSupportedLocale(fallback) {
		return fallback
	}
	return "zh-CN"
}

// ErrKeyNotFound 保留错误供测试使用
var ErrKeyNotFound = errors.New("i18n key not found")
