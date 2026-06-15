# Dujiao-Next Telegram Bot

[![CI](https://github.com/dujiao-next/telegram-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/dujiao-next/telegram-bot/actions/workflows/ci.yml)

与 [dujiao-next](https://github.com/dujiao-next) 后端配套的 Telegram Bot 客户端实现。

## 特性

- 🤖 通过 dujiao-next 的 Channel API（HMAC-SHA256 签名）调用后端业务能力
- 🛍️ 浏览商品 / 下单 / 支付 / 钱包 / 礼品卡 / 推广返利，覆盖前台用户主要场景
- 🎨 多语言：内置 `zh-CN / zh-TW / en-US` 三种语言，i18n 资源以 `//go:embed` 嵌入
- ⚙️ 菜单、欢迎语、帮助中心均由后台 `/admin/settings/telegram-bot` 配置，Bot 实时拉取
- 📡 接收 dujiao 主动推送：`order_fulfilled` / `order_paid` / `wallet_recharge_succeeded`
- 💓 心跳上报连接状态、Bot 版本、机器码、授权状态
- 📦 **单二进制发布**：`CGO_ENABLED=0` 静态链接，无运行时依赖

## 部署

### 1. 后台创建「渠道客户端」

1. 登录 dujiao-next 管理面板
2. 进入「渠道客户端管理」
3. 新建一个客户端：
   - **Name**：`tg-shop-prod`
   - **Channel Type**：`telegram_bot`
   - **Bot Token**：在 [@BotFather](https://t.me/BotFather) 申请的 Bot Token
   - **Callback URL**：Bot 的对外公网地址，例如 `https://bot.example.com`
   - **Description**：可选
4. **创建后立即保存页面返回的 Channel Key / Channel Secret**（只显示一次）

### 2. 后台配置 Bot 设置

进入「Telegram Bot 设置」配置：
- 基础信息（名称、描述、封面图、支持链接）
- 默认语言
- 欢迎消息（可按语言维护）
- 自定义菜单（可设置动作类型 `builtin / url / web_app / command`）
- 帮助中心主题

### 3. 启动 Bot 二进制

```bash
# 上传 binary
scp telegram-bot user@bot-server:/opt/telegram-bot/

# 上传 config
scp configs/config.example.yaml user@bot-server:/opt/telegram-bot/configs/config.yaml

# 编辑 /opt/telegram-bot/configs/config.yaml 填好 api.base_url

# 注入渠道凭证（必填）
export TG_CHANNEL_KEY="从后台复制的 ChannelKey"
export TG_CHANNEL_SECRET="从后台复制的 ChannelSecret"

# 启动
/opt/telegram-bot/telegram-bot -config /opt/telegram-bot/configs/config.yaml
```

### 4. 公网回调

Bot 监听的 `:8444` 端口必须能从 dujiao-next 服务访问。后台填写的 `CallbackURL` 形如：

```
https://bot.example.com
```

dujiao 会拼接 `/internal/order-fulfilled` 等路径 POST 进来。同样的 HMAC 验签由 Bot 端 `internal/notify` 完成。

## 配置参考

```yaml
# configs/config.yaml
server:
  listen: ":8444"
  public_base_url: ""            # 可选，便于自检日志
  max_body_bytes: 1048576        # HTTP body 最大字节数（默认 1MB）

api:
  base_url: "https://shop.example.com"
  timeout_seconds: 15
  insecure_skip_verify: false    # 仅在内网自签证书时打开
  max_idle_conns_per_host: 16

bot:
  machine_code: ""               # 留空自动取 hostname
  bot_version: "1.0.0"
  webhook_status: "active"
  license_status: "active"       # active / expired / revoked / suspended
  license_expires_at: ""         # RFC3339，如 2026-12-31T23:59:59Z
  default_locale: "zh-CN"        # zh-CN / zh-TW / en-US
  default_currency: "CNY"        # 默认货币代码
  update_interval_seconds: 25    # 心跳（须 < 60s）
  config_reload_seconds: 60      # 配置重拉
  poll_timeout_seconds: 10
  poll_limit: 100
  product_page_size: 5
  order_page_size: 5
  wallet_page_size: 10
  affiliate_page_size: 8
  quantity_options: [1, 2, 5, 10]

log:
  level: "info"
```

环境变量（敏感配置）：

| 变量 | 必填 | 说明 |
|---|---|---|
| `TG_CHANNEL_KEY` | ✅ | 后台创建客户端时返回 |
| `TG_CHANNEL_SECRET` | ✅ | 后台创建客户端时返回 |
| `TG_API_BASE_URL` | | 覆盖 `api.base_url` |
| `TG_BOT_VERSION` | | 覆盖 `bot.bot_version` |
| `TG_MACHINE_CODE` | | 覆盖 `bot.machine_code` |
| `TG_DEFAULT_LOCALE` | | 覆盖 `bot.default_locale` |
| `TG_DEFAULT_CURRENCY` | | 覆盖 `bot.default_currency` |
| `TG_LICENSE_STATUS` | | 覆盖 `bot.license_status` |
| `TG_LICENSE_EXPIRES_AT` | | 覆盖 `bot.license_expires_at`（RFC3339） |
| `TG_LISTEN` | | 覆盖 `server.listen` |

## 内置菜单（7 项）

与 dujiao-next `internal/service/telegram_bot_setting.go:builtinMenuKeysOrder` 完全对齐：

| Key | 默认文案（zh-CN） | 行为 |
|---|---|---|
| `shop_home` | 🛍️ 开始购物 | 浏览分类 → 商品 → 下单 |
| `my_orders` | 📦 我的订单 | 列表 → 详情 → 支付 / 取消 |
| `my_wallet` | 💰 我的钱包 | 余额 / 流水 / 充值 / 礼品卡入口 |
| `affiliate` | 📣 推广返利 | 开通 / 概览 / 佣金 / 提现 |
| `gift_card` | 🎁 礼品卡兑换 | 输入兑换码 → 钱包到账 |
| `switch_language` | 🌐 切换语言 | zh-CN / zh-TW / en-US |
| `contact_support` | ❓ 帮助中心 | 帮助中心 + 客服链接 |

后台菜单编辑器可调整这些项的 label、order、enabled、action.type / action.value（最多 20 项）。

## 构建

```bash
# 单平台
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/telegram-bot ./cmd/telegram-bot

# Docker
docker build -t telegram-bot:latest .

# 多平台（需要 Go 1.22+ 和 goreleaser）
goreleaser build --snapshot --clean

# 正式发版
git tag v1.0.0
goreleaser release --clean
```

## 项目结构

```
telegram-bot/
├── cmd/telegram-bot/main.go        # 入口
├── internal/
│   ├── config/                     # YAML 加载
│   ├── crypto/                     # HMAC-SHA256 签名
│   ├── api/                        # Channel API 客户端
│   ├── bot/                        # telebot 主体（菜单/handler/i18n 集成）
│   ├── notify/                     # BotNotify HTTP 接收服务
│   ├── i18n/                       # 多语言资源（embed）
│   ├── state/                      # 进程内 user session
│   ├── formatter/                  # TG HTML/金额/状态文案
│   └── logger/                     # zap logger
├── configs/config.example.yaml
├── .goreleaser.yaml
└── go.mod
```

## 协议对齐

与 dujiao-next 完全兼容的接口：

| Bot 调用 | 后端 Handler |
|---|---|
| `GET /api/v1/channel/telegram/config` | `channel_telegram_bot.go:GetBotConfig` |
| `POST /api/v1/channel/telegram/heartbeat` | `channel_telegram_bot.go:ReportHeartbeat` |
| `POST /api/v1/channel/identities/telegram/{resolve,provision,bind}` | `channel_identity.go` |
| `GET /api/v1/channel/catalog/{categories,products,products/:id}` | `channel_catalog.go` |
| `GET /api/v1/channel/member-levels` | `channel_catalog.go` |
| `POST /api/v1/channel/orders/{preview,POST}` | `channel_order.go` |
| `GET /api/v1/channel/orders{,/:id,/by-order-no/:no}` | `channel_order.go` |
| `POST /api/v1/channel/orders/:id/cancel` | `channel_order.go` |
| `GET /api/v1/channel/payment-channels` | `channel_order.go` |
| `POST /api/v1/channel/payments` | `channel_order.go` |
| `GET /api/v1/channel/payments/{latest,:id}` | `channel_order.go` |
| `GET /api/v1/channel/wallet{,/transactions}` | `channel_wallet.go` |
| `POST /api/v1/channel/wallet/{recharge,gift-card/redeem}` | `channel_wallet.go` |
| `POST /api/v1/channel/affiliate/{click,open,withdraws}` | `channel_affiliate.go` |
| `GET /api/v1/channel/affiliate/{dashboard,commissions,withdraws}` | `channel_affiliate.go` |

Bot 端实现的回调端点（接收 dujiao asynq worker 推送）：

| Path | 事件 | 来源 |
|---|---|---|
| `POST /internal/order-fulfilled` | 订单交付 | `fulfillment_service.go:NotifyBotOrderFulfilled` |
| `POST /internal/order-paid` | 订单支付成功 | `payment_service_callback.go:enqueueOrderPaidBotNotifyAsync` |
| `POST /internal/wallet-recharge-succeeded` | 钱包充值成功 | `payment_service_callback.go:enqueueWalletRechargeBotNotifyAsync` |

所有请求均使用与 `internal/upstream/signer.go` 一致的 HMAC-SHA256 签名算法（`signString = "{method}\n{path}\n{timestamp}\n{body_md5}"`），时间戳偏差上限 60 秒。

## License

本项目独立于 dujiao-next，仅作客户端实现参考。请遵守 dujiao-next 项目的 GPL-3.0 协议。
