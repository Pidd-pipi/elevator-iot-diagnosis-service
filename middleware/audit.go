package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// auditWriter 包装 ResponseWriter，捕获响应状态码与字节数。
type auditWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader 捕获状态码。
func (w *auditWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write 累计写入字节数。
func (w *auditWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush 支持 http.Flusher（SSE/流式响应场景）。
func (w *auditWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AuditLogger 审计日志中间件：
//  1. 为每个请求输出访问日志（方法、路径、状态、耗时）；
//  2. 对写请求（非 GET/HEAD/OPTIONS）在审计仓储中留痕，
//     触达 handler → middleware → audit store 全链路。
func AuditLogger(st *store.Store, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			aw := &auditWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(aw, r)
			duration := time.Since(start)

			logger.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", aw.status,
				"bytes", aw.bytes,
				"duration_ms", duration.Milliseconds(),
				"request_id", r.Header.Get("X-Request-Id"),
				"remote", r.RemoteAddr,
			)

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				return
			}
			// 写请求留痕到审计仓储。
			log := domain.NewAuditLog(
				"http.request",
				actorHeader(r),
				"http",
				r.URL.Path,
				"HTTP "+r.Method+" "+r.URL.Path+" -> "+http.StatusText(aw.status),
				start,
			)
			log.ID = store.NewID("audit")
			log.RequestID = r.Header.Get("X-Request-Id")
			st.Audits.Append(log)
		})
	}
}

// actorHeader 从请求头提取操作者，缺省为 console。
func actorHeader(r *http.Request) string {
	if a := r.Header.Get("X-Actor"); a != "" {
		return a
	}
	return "console"
}
