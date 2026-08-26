// Package middleware 提供跨切面的 HTTP 中间件：
// 请求 trace id、审计日志、统一错误/panic 处理。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// ctxKey 用于在 context 中存取请求级数据。
type ctxKey int

const (
	// requestIDKey 存放 trace id。
	requestIDKey ctxKey = iota
)

// RequestID 注入中间件：为每个请求生成/透传 trace id，
// 并写入响应头 X-Request-Id。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newTraceID()
		}
		w.Header().Set("X-Request-Id", rid)
		next.ServeHTTP(w, r)
	})
}

// RequestIDFrom 从 context 中取出 trace id。
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// newTraceID 生成 16 字节随机十六进制 trace id。
func newTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return hex.EncodeToString(buf)
}
