// Package crypto 提供与 dujiao-next 一致的 HMAC-SHA256 签名算法。
//
// 源码同源于 dujiao-next/internal/upstream/signer.go，便于 Bot 与后端
// 在不同 module 下保持签名串构造完全一致：
//
//	signString = "{method}\n{path}\n{timestamp}\n{body_md5}"
//
// 渠道 API 在请求头携带：
//   - Dujiao-Next-Channel-Key
//   - Dujiao-Next-Channel-Timestamp
//   - Dujiao-Next-Channel-Signature
//
// Bot 端同时使用本工具校验 dujiao 主动回调（BotNotify）请求的签名。
package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	// HeaderChannelKey 渠道客户端 Key 头
	HeaderChannelKey = "Dujiao-Next-Channel-Key"
	// HeaderTimestamp 渠道 API 时间戳头
	HeaderTimestamp = "Dujiao-Next-Channel-Timestamp"
	// HeaderSignature 渠道 API 签名头
	HeaderSignature = "Dujiao-Next-Channel-Signature"

	// MaxTimestampSkew 最大时间戳偏差（秒）
	MaxTimestampSkew = 60
)

// Sign 生成 HMAC-SHA256 签名
func Sign(secret, method, path string, timestamp int64, body []byte) string {
	bodyMD5 := md5Hex(body)
	signString := fmt.Sprintf("%s\n%s\n%d\n%s", method, path, timestamp, bodyMD5)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signString))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 验证签名
func Verify(secret, method, path, signature string, timestamp int64, body []byte) bool {
	expected := Sign(secret, method, path, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// IsTimestampValid 检查时间戳是否在有效范围内
func IsTimestampValid(timestamp int64) bool {
	now := time.Now().Unix()
	return math.Abs(float64(now-timestamp)) <= MaxTimestampSkew
}

// ParseTimestamp 解析时间戳字符串
func ParseTimestamp(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func md5Hex(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
