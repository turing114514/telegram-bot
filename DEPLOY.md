# Dujiao-Next Telegram Bot 部署文档

本文档覆盖从零搭建一个生产可用的 Telegram Bot 客户端，并接入 dujiao-next 后端的完整流程。

---

## 0. 部署拓扑

```
┌────────────────────┐     HMAC 签名      ┌────────────────────┐
│                    │ ───────────────►   │                    │
│   dujiao-next      │  Channel API       │   telegram-bot     │
│   (主站)           │ ◄───────────────   │   (本服务)         │
│                    │  BotNotify POST    │                    │
└────────────────────┘                    └─────────┬──────────┘
                                                     │
                                          long polling / WebApp
                                                     │
                                                     ▼
                                              ┌────────────┐
                                              │  Telegram  │
                                              │  用户端    │
                                              └────────────┘
```

**网络要求**

| 方向 | 端口 | 说明 |
|---|---|---|
| telegram-bot → dujiao-next | 出站 HTTPS 443 | 调用 Channel API（`/api/v1/channel/*`） |
| dujiao-next → telegram-bot | 入站 HTTPS 443（建议） | BotNotify 主动回调 `/internal/*` |
| telegram-bot → Telegram | 出站 HTTPS 443 | Bot polling / sendMessage |

公网可达是关键——若 dujiao-next 与 Bot 不在同一内网，必须给 Bot 配公网域名或反代。

---

## 1. 后端准备（在 dujiao-next 管理面板操作）

### 1.1 创建渠道客户端

进入 **渠道客户端管理 → 新建**：

| 字段 | 填写示例 |
|---|---|
| Name | `tg-shop-prod` |
| Channel Type | `telegram_bot` |
| Bot Token | `123456:ABC-...`（从 @BotFather 申请） |
| Callback URL | `https://bot.example.com`（Bot 服务的对外地址，无尾斜杠） |
| Description | 生产环境 Bot |

**点击保存后立刻复制页面返回的 Channel Key 和 Channel Secret。** 它们仅在创建时明文显示一次，丢失需重置。

### 1.2 配置 Bot 设置

进入 **Telegram Bot 设置**：

- **启用** ✅
- **默认语言**：`zh-CN` / `zh-TW` / `en-US`（与目标用户群匹配）
- **基础信息**：
  - 展示名称
  - 描述（按语言填）
  - 封面图 URL
  - 支持链接
- **欢迎消息**：按语言维护文案
- **菜单**：按需增删，7 个内置项的 key 固定为 `shop_home / my_orders / my_wallet / affiliate / gift_card / switch_language / contact_support`
- **帮助中心**：可添加多个主题，每个主题的 `key / enabled / order / summary / title / content` 都按语言维护

后台保存后，`config_version` 自动 +1，Bot 会在下一次轮询（默认 60s）拉取新配置并自动更新菜单。

---

## 2. 服务器环境

### 2.1 硬件最低要求

- CPU：1 核
- 内存：256 MB
- 磁盘：200 MB（仅二进制 + 日志）
- 系统：Linux（推荐 Debian 12 / Ubuntu 22.04+）/ macOS / Windows

### 2.2 网络

- **必须**能访问 Telegram API（`api.telegram.org`），否则 bot polling 起不来
- **必须**能访问 dujiao-next API base URL
- **必须**能从 dujiao-next 服务访问 Bot 的回调端口（公网或内网放通）

### 2.3 反向代理（强烈推荐）

Bot 默认监听 `:8444` 暴露 HTTP，但生产建议放在 nginx / caddy 后面提供 HTTPS / 静态资源 / 限流。

---

## 3. 部署方式

### 3.1 方式 A：单二进制直接部署（最简单）

#### 下载预编译产物

从 GitHub Releases 下载对应平台包：

```bash
# 以 linux/amd64 为例
curl -L -o telegram-bot.tar.gz \
  https://github.com/dujiao-next/telegram-bot/releases/latest/download/telegram-bot_1.0.0_linux_amd64.tar.gz
tar -xzf telegram-bot.tar.gz
cd telegram-bot_*
ls
# 预期看到：telegram-bot  README.md  configs/
```

#### 或自行编译

需要 Go 1.22+：

```bash
git clone https://github.com/dujiao-next/telegram-bot.git
cd telegram-bot
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=v1.0.0" \
  -o dist/telegram-bot ./cmd/telegram-bot
```

#### 准备配置

```bash
sudo mkdir -p /opt/telegram-bot
sudo cp dist/telegram-bot /opt/telegram-bot/
sudo cp configs/config.example.yaml /opt/telegram-bot/configs/config.yaml
sudo chmod 755 /opt/telegram-bot/telegram-bot
sudo chmod 640 /opt/telegram-bot/configs/config.yaml
```

编辑 `/opt/telegram-bot/configs/config.yaml`：

```yaml
server:
  listen: "127.0.0.1:8444"          # 仅监听本地，由反代转发
  public_base_url: ""               # 可选

api:
  base_url: "https://shop.example.com"
  timeout_seconds: 15
  insecure_skip_verify: false      # 私签证书才置 true

bot:
  machine_code: ""                  # 留空取 hostname
  bot_version: "1.0.0"
  webhook_status: "active"
  default_locale: "zh-CN"
  update_interval_seconds: 25       # 心跳，须 < 60
  config_reload_seconds: 60
  poll_timeout_seconds: 10
  poll_limit: 100

log:
  level: "info"
```

设置渠道凭证（**推荐环境变量，也可写入 `configs/config.yaml` 的 `channel` 节**）：

```bash
sudo tee /opt/telegram-bot/.env > /dev/null <<'EOF'
TG_CHANNEL_KEY=<从后台复制的 ChannelKey>
TG_CHANNEL_SECRET=<从后台复制的 ChannelSecret>
EOF
sudo chmod 600 /opt/telegram-bot/.env
```

如选择写入 YAML：

```yaml
channel:
  key: <ChannelKey>
  secret: <ChannelSecret>
```

> ⚠️ 写在 YAML 里时，务必确保 `config.yaml` 不被提交到 git，且文件权限设置为 `600`。启动日志会打印一条警告提醒。

#### systemd 托管

`/etc/systemd/system/telegram-bot.service`：

```ini
[Unit]
Description=Dujiao-Next Telegram Bot
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tgbot
Group=tgbot
WorkingDirectory=/opt/telegram-bot
ExecStart=/opt/telegram-bot/telegram-bot -config /opt/telegram-bot/configs/config.yaml
Restart=always
RestartSec=5
EnvironmentFile=/opt/telegram-bot/.env
LimitNOFILE=65536

# 资源限制（防止 leak）
MemoryMax=512M
TasksMax=2048

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=telegram-bot

# 加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/telegram-bot /var/log/telegram-bot

[Install]
WantedBy=multi-user.target
```

启用：

```bash
sudo useradd -r -s /usr/sbin/nologin tgbot
sudo mkdir -p /var/log/telegram-bot
sudo chown tgbot:tgbot /var/log/telegram-bot
sudo chown -R tgbot:tgbot /opt/telegram-bot
sudo systemctl daemon-reload
sudo systemctl enable --now telegram-bot
sudo systemctl status telegram-bot
sudo journalctl -u telegram-bot -f
```

### 3.2 方式 B：Docker 部署

#### 镜像

发布版本会自动推送到 GitHub Container Registry：

```bash
ghcr.io/dujiao-next/telegram-bot:<version>
```

也可本地构建：

```bash
docker build -t telegram-bot:1.0.0 .
```

#### docker compose

`docker-compose.yml` 已提供，先准备配置：

```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml，填好 api.base_url

echo "TG_CHANNEL_KEY=xxx" > .env
echo "TG_CHANNEL_SECRET=yyy" >> .env

docker compose up -d
docker compose logs -f
```

容器内默认读取 `/etc/telegram-bot/config.yaml`，`wget` 已内置用于 healthcheck。

### 3.3 方式 C：Nginx 反向代理（HTTPS）

Bot 监听 `127.0.0.1:8444`，由 nginx 提供 HTTPS / 限流：

```nginx
upstream telegram_bot {
    server 127.0.0.1:8444;
    keepalive 16;
}

server {
    listen 443 ssl http2;
    server_name bot.example.com;

    ssl_certificate     /etc/letsencrypt/live/bot.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/bot.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # BotNotify 三个端点必须转发（保持路径原样）
    location /internal/ {
        proxy_pass http://telegram_bot;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # dujiao 端会带 Dujiao-Next-Channel-* 头
        proxy_pass_request_headers on;
        # 长 body / 心跳超时
        proxy_read_timeout 30s;
        proxy_send_timeout 30s;
    }

    location /healthz {
        proxy_pass http://telegram_bot;
        access_log off;
    }

    # 限流：每秒 5 个 BotNotify（防滥用）
    limit_req_zone $binary_remote_addr zone=botnotify:10m rate=5r/s;
    limit_req zone=botnotify burst=10 nodelay;
}
```

> ⚠️ dujiao worker 在签名校验时使用 **URL.Path** 作为签名字段，nginx 反代时**不能修改 path**（默认行为就是不改）。如要加前缀请同步修改 `internal/notify/server.go:route` 与 `internal/worker/asynq_worker.go:path` 双方。

放通端口：

```bash
sudo ufw allow 443/tcp
sudo ufw reload
```

---

## 4. 健康检查

Bot 在 `:8444/healthz` 提供探活：

```bash
curl https://bot.example.com/healthz
# ok
```

接入你的监控系统：

| 系统 | 探活命令 |
|---|---|
| systemd | `Restart=always` + `WatchdogSec=60` |
| Docker | 见上方 `healthcheck` 段 |
| K8s liveness | `httpGet: /healthz, port: 8444` |
| Prometheus blackbox | `prober.http.get: https://bot.example.com/healthz` |

**心跳也是健康指标**——后台 `Telegram Bot → 状态` 面板会显示 `connected` / `last_seen_at` / `bot_version`，如果发现 `connected=false` 持续超过 2 个心跳周期（>50s），意味着 Bot 进程异常或网络问题。

---

## 5. 监控与日志

### 5.1 日志

Bot 输出 JSON 结构化日志到 stdout（systemd 会自动接 journald）：

```bash
# 实时查看
sudo journalctl -u telegram-bot -f

# 按时间范围
sudo journalctl -u telegram-bot --since "1 hour ago"

# 按级别
sudo journalctl -u telegram-bot -p warning

# 导出
sudo journalctl -u telegram-bot --since today > /tmp/bot.log
```

日志字段：
- `level` / `ts` / `caller`（自动）
- `msg` + 自定义 keys（`error / order_id / telegram_user_id / config_version ...`）

### 5.2 后台可观测性

dujiao-next 后台「Telegram Bot → 状态」面板会展示：

- 连接状态（是否在线）
- 最近心跳时间
- Bot 版本
- Webhook 状态
- 机器码（多实例时区分）
- 授权状态 / 告警（如果对接了授权系统）
- config_version（配置同步是否跟上）

「渠道客户端管理」可看每个 client 的 `last_used_at`，判断是否被 Bot 真实使用。

### 5.3 关键告警

建议接入告警系统（AlertManager / Grafana / 自家 IM）：

| 指标 | 阈值 | 含义 |
|---|---|---|
| Bot 进程退出 | 立即 | systemd 自动重启，但仍要排查根因 |
| 心跳缺失 > 90s | 告警 | Bot 与后端链路故障（防火墙 / Token 失效 / 配置变更） |
| `/healthz` 5xx 持续 > 1m | 告警 | 端口监听失败 |
| 渠道 API 调用 4xx 比例 > 10% | 警告 | ChannelKey/Secret 错误或过期 |
| 5xx 比例 > 5% | 告警 | dujiao 后端故障 |
| 日志中含 `signature verify failed` | 立即 | 有人在伪造回调 |

---

## 6. 升级与回滚

### 6.1 升级

```bash
# 1. 拉取新版本
cd /opt/telegram-bot
sudo systemctl stop telegram-bot
sudo cp telegram-bot telegram-bot.bak-$(date +%Y%m%d)
# 覆盖二进制
sudo cp /path/to/new/telegram-bot .

# 2. 检查配置是否有新字段（阅读 release notes）
diff -u configs/config.yaml /path/to/new/configs/config.example.yaml

# 3. 启动并观察
sudo systemctl start telegram-bot
sudo journalctl -u telegram-bot -f
```

### 6.2 回滚

```bash
sudo systemctl stop telegram-bot
sudo cp telegram-bot.bak-YYYYMMDD telegram-bot
sudo systemctl start telegram-bot
```

### 6.3 配置重置密钥（不重启 Bot 的情况）

若需要重置 Channel Secret 而 Bot 进程仍要继续运行：

```bash
# 1. 后台「渠道客户端 → 重置 Secret」，复制新 Secret
# 2. 更新 .env
sudo systemctl edit telegram-bot
# 在 [Service] 下加：
#   Environment="TG_CHANNEL_SECRET=new_secret"
sudo systemctl restart telegram-bot
```

> 注意：单实例部署必须重启。多实例 + 配置中心可滚动重启。

---

## 7. 故障排查

### 7.1 Bot 启动后立刻退出

```bash
sudo journalctl -u telegram-bot -n 50
```

常见错误：
- `load config: open /opt/.../config.yaml: no such file` → 检查 `-config` 路径
- `api: channel key is required` → 没设 `channel.key` 或 `TG_CHANNEL_KEY`
- `api: base url is required` → `api.base_url` 为空
- `bot: cannot start without initial bot config` → 拉不到 `bot_token`，检查：
  - 后台是否创建了 `channel_type=telegram_bot` 的客户端
  - Bot Token 是否为空
  - 后端防火墙是否允许 Bot 出口

### 7.2 后台显示 `connected=false`

- 25s 心跳：若 2 周期仍 false，说明网络/鉴权问题
- 看后端日志：搜索 `channel_auth_error` 关键词
- 鉴权失败：检查 `TG_CHANNEL_KEY / TG_CHANNEL_SECRET` 与后台一致
- 时间戳偏差：服务器时区/时间差过大，调整 NTP

### 7.3 Telegram 用户发消息无响应

- Bot Token 是否有效（在 @BotFather 用 `/token` 验证）
- 网络能否访问 `api.telegram.org`：`curl https://api.telegram.org`
- 日志中 `telebot polling started` 是否出现
- 检查后台「渠道客户端」`last_used_at` 是否在变化

### 7.4 BotNotify 不触发

- 确认后台「Telegram Bot 设置 → 启用」✅
- 确认 Callback URL 与 Bot 实际可访问地址一致
- curl 测试回调：
  ```bash
  curl -X POST https://bot.example.com/internal/order-fulfilled \
    -H "Content-Type: application/json" \
    -H "Dujiao-Next-Channel-Key: $KEY" \
    -H "Dujiao-Next-Channel-Timestamp: $(date +%s)" \
    -H "Dujiao-Next-Channel-Signature: <sig>" \
    -d '{"order_id":1,"telegram_user_id":"123"}'
  ```
- Bot 日志中是否出现 `signature verify failed`
- 时钟同步：`chronyc tracking`

### 7.5 群发 / 支付 / 订单报错

绝大多数都是后端业务规则触发——按 `error_code` 在 dujiao-next `internal/i18n/locales/zh-CN.json` 搜索对应的错误文案。Bot 已自动透传后端 `Msg` 字段给用户。

---

## 8. 安全建议

1. **Channel Secret 推荐用环境变量**，避免写进 `config.yaml`；如必须写入，请确保文件权限 `600` 且不提交到 git。
2. **`.env` 文件权限 600**，仅 `tgbot` 用户可读
3. **Bot Token 视为密码**：泄露后立刻去 @BotFather `/revoke` 并更新后台
4. **CallbackURL 限制来源**：nginx `allow / deny` 仅放通 dujiao 服务器 IP 段
5. **TLS 证书**：Let's Encrypt 3 个月自动续期，监控到期
6. **不要把 `bot_token` 写进日志**：本 Bot 设计上不记录 token 内容（`maskBotToken` 脱敏），但要警惕后端 push 的 `bot_token` 字段
7. **更新及时**：goreleaser 出新版时关注 [GitHub Releases](https://github.com/dujiao-next/telegram-bot/releases)

---

## 9. 多实例部署

需要水平扩展时：

- 每个实例的 `machine_code` 必须不同（默认会用 hostname，多实例天然满足）
- 后台 `ChannelClient` 的 `last_used_at` 会被最近调用的实例更新——通过 `machine_code` 区分
- BotNotify 推送**不重复**（一个 chat_id 只发一次），但多实例时需在 Bot 侧做幂等：建议用 Redis 记录 `(telegram_user_id, order_id)` 已处理集合
- 当前内置 `state.Manager` 是进程内 `sync.Map`，多实例间**不共享**用户 session。可替换为 Redis：

  ```go
  // internal/state/state.go 把 Manager 替换为：
  // - GET  user:{tgID}:session  (TTL 1h)
  // - SET  user:{tgID}:session
  ```

---

## 10. 部署检查清单

- [ ] 后台已创建 `telegram_bot` 类型渠道客户端，状态启用
- [ ] 已保存 ChannelKey / ChannelSecret 至安全位置
- [ ] 后台「Telegram Bot 设置」已启用并配置完毕
- [ ] `config.yaml` 中 `api.base_url` 正确指向 dujiao-next
- [ ] `channel.key / channel.secret` 或 `TG_CHANNEL_KEY / TG_CHANNEL_SECRET` 已注入
- [ ] `CallbackURL` 与 Bot 实际公网地址一致
- [ ] nginx / 反代已配置 HTTPS（如启用）
- [ ] systemd / docker 开机自启已配置
- [ ] `/healthz` 探活正常
- [ ] 后台「状态」面板显示 `connected=true`
- [ ] 在 Telegram 中给 Bot 发 `/start` 看到菜单
- [ ] 完成一笔测试订单，观察 BotNotify 推送
- [ ] 日志告警通道已配置
- [ ] `.env` 权限 600
- [ ] 备份当前 binary 与 config

---

## 附录 A：完整 release 包内容

```text
telegram-bot_1.0.0_linux_amd64.tar.gz
├── telegram-bot              # 单二进制（~9.9MB）
├── README.md
├── configs/
│   └── config.example.yaml
└── LICENSE
```

## 附录 B：环境变量速查

| 变量 | 必填 | YAML 等价项 | 默认 | 说明 |
|---|---|---|---|---|
| `TG_CHANNEL_KEY` | ✅* | `channel.key` | — | 后台渠道客户端 ChannelKey |
| `TG_CHANNEL_SECRET` | ✅* | `channel.secret` | — | 后台渠道客户端 ChannelSecret |
| `TG_API_BASE_URL` | | `api.base_url` | — | API 基础 URL |
| `TG_BOT_VERSION` | | `bot.bot_version` | `1.0.0` | 上报的 Bot 版本 |
| `TG_MACHINE_CODE` | | `bot.machine_code` | hostname | 上报的机器码 |
| `TG_DEFAULT_LOCALE` | | `bot.default_locale` | `zh-CN` | 默认语言 |
| `TG_LISTEN` | | `server.listen` | `:8444` | HTTP 监听地址 |

\* 必填但可写入 YAML；环境变量优先级高于 YAML。

## 附录 C：常用命令速查

```bash
# 启动 / 停止 / 重启
sudo systemctl {start,stop,restart,status} telegram-bot

# 日志
sudo journalctl -u telegram-bot -f

# 升级
sudo systemctl stop telegram-bot
sudo install -m 0755 new-telegram-bot /opt/telegram-bot/telegram-bot
sudo systemctl start telegram-bot

# 回滚
sudo systemctl stop telegram-bot
sudo install -m 0755 telegram-bot.bak-YYYYMMDD /opt/telegram-bot/telegram-bot
sudo systemctl start telegram-bot

# 查看版本
/opt/telegram-bot/telegram-bot -version

# 配置自检
/opt/telegram-bot/telegram-bot -config /opt/telegram-bot/configs/config.yaml
# （应能看到 fatal: channel key/secret are required，确认凭证已正确注入）
```
