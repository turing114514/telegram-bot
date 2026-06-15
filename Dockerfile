# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /src

# 安装 git（go mod 可能需要）
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /bin/telegram-bot ./cmd/telegram-bot

# 运行阶段：使用 Alpine，方便挂载配置和调试
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /bin/telegram-bot /usr/local/bin/telegram-bot
COPY --from=builder /src/configs/config.example.yaml /etc/telegram-bot/config.example.yaml

# 默认挂载 /etc/telegram-bot/config.yaml 并注入环境变量运行
ENTRYPOINT ["telegram-bot"]
CMD ["-config", "/etc/telegram-bot/config.yaml"]
