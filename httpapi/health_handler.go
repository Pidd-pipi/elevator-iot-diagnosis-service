package httpapi

import (
	"net/http"
	"time"
)

// HealthHandler 健康检查与就绪检查。
type HealthHandler struct {
	startedAt time.Time
}

// NewHealthHandler 构造健康检查处理器。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{startedAt: time.Now()}
}

// healthPayload 返回统一的健康/就绪检查载荷。
func (h *HealthHandler) healthPayload() map[string]any {
	return map[string]any{
		"status":     "ok",
		"uptime_sec": int(time.Since(h.startedAt).Seconds()),
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

// Health GET /healthz 与 GET /api/healthz（存活检查）。
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	OK(w, r, h.healthPayload())
}

// Ready GET /readyz（就绪检查）。当前无外部依赖，与健康检查等价。
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	OK(w, r, h.healthPayload())
}
