// Package logger 提供 zap sugar logger 的初始化。
package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 包装 zap.SugaredLogger 以提供 Fatalw
type Logger struct {
	*zap.SugaredLogger
}

// Fatalw 记录并以状态码 1 退出
func (l *Logger) Fatalw(msg string, keysAndValues ...any) {
	l.SugaredLogger.Fatalw(msg, keysAndValues...)
}

// Init 初始化 logger
func Init(level string) (*Logger, error) {
	lvl := zapcore.InfoLevel
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn", "warning":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.DisableStacktrace = true
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{SugaredLogger: l.Sugar()}, nil
}

