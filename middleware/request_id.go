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
// 写入响应头 X-Request-Id，并存入请求 context 与入站请求头，
// 使下游 handler、审计日志、统一响应体都能稳定取到同一个 id。
//
// 同时回写 r.Header 是关键：审计中间件（middleware/audit.go）与统一响应
// （httpapi/response.go）均通过 r.Header.Get("X-Request-Id") 取值，
// 仅写入 context 会让走 header 读取路径的调用方拿到空串。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newTraceID()
		}
		// 回写入站请求头，供依赖 r.Header 的下游（审计/响应体）读取。
		r.Header.Set("X-Request-Id", rid)
		// 同步写入响应头与 context，供依赖 context 的下游（panic 恢复/错误响应）读取。
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
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
