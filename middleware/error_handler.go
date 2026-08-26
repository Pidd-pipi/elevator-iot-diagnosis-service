package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// errorResponse 与 httpapi 统一响应格式保持一致。
type errorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// Recoverer 统一 panic 恢复中间件：捕获 handler 中的 panic，
// 记录堆栈并返回 500 统一格式错误，避免进程崩溃。
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"path", r.URL.Path,
						"request_id", RequestIDFrom(r.Context()),
						"stack", string(debug.Stack()))
					writeError(w, r, http.StatusInternalServerError, 50000, "服务器内部错误，请稍后重试")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// writeError 输出统一错误响应。
func writeError(w http.ResponseWriter, r *http.Request, status, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if rid := RequestIDFrom(r.Context()); rid != "" {
		w.Header().Set("X-Request-Id", rid)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFrom(r.Context()),
	})
}
