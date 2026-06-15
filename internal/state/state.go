// Package state 提供 Bot 端用户会话状态管理。
//
// 当前实现为进程内 sync.Map（重启即清空），单实例部署足够。
// 升级为 Redis 时，把 Manager 替换为 redis 客户端并加 TTL 即可。
package state

import (
	"sync"
	"time"
)

// Session 单个用户的会话状态
type Session struct {
	Locale    string    // 用户偏好语言（zh-CN / zh-TW / en-US）
	UpdatedAt time.Time // 最近一次更新时间

	// 购物流程临时数据
	BrowsingCategoryID uint    // 当前正在浏览的分类
	LastProductID      uint    // 最近查看的商品 ID
	LastProductQty     int     // 最近选中的数量
	PendingOrderItems  []OrderItemDraft

	// 充值 / 提现临时数据
	RechargePendingAmount string
	WithdrawPendingAmount string
	WithdrawPendingCh     string

	// 礼品卡 / 优惠码临时输入
	AwaitingGiftCard bool
	AwaitingCoupon   bool
	PendingCoupon    string
}

// OrderItemDraft 订单草稿项
type OrderItemDraft struct {
	ProductID       uint
	SKUID           uint
	Quantity        int
	FulfillmentType string
}

// Manager 会话管理器
type Manager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session // key = Telegram user ID
}

// NewManager 构造会话管理器
func NewManager() *Manager {
	return &Manager{sessions: make(map[int64]*Session)}
}

// Get 获取会话（不存在时自动创建）
func (m *Manager) Get(tgUserID int64) *Session {
	m.mu.RLock()
	s, ok := m.sessions[tgUserID]
	m.mu.RUnlock()
	if ok {
		return s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok = m.sessions[tgUserID]; ok {
		return s
	}
	s = &Session{UpdatedAt: time.Now()}
	m.sessions[tgUserID] = s
	return s
}

// Set 整体覆盖
func (m *Manager) Set(tgUserID int64, s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.UpdatedAt = time.Now()
	m.sessions[tgUserID] = s
}

// Touch 标记活跃
func (m *Manager) Touch(tgUserID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[tgUserID]; ok {
		s.UpdatedAt = time.Now()
	}
}

// SetLocale 设置偏好语言
func (m *Manager) SetLocale(tgUserID int64, locale string) {
	s := m.Get(tgUserID)
	s.Locale = locale
	m.Touch(tgUserID)
}

// GetLocale 读取偏好语言（空则回退到 caller 给的 default）
func (m *Manager) GetLocale(tgUserID int64, fallback string) string {
	s := m.Get(tgUserID)
	if s.Locale != "" {
		return s.Locale
	}
	return fallback
}

// Reset 清空指定用户的会话
func (m *Manager) Reset(tgUserID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tgUserID)
}
