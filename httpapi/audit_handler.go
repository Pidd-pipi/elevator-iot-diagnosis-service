package httpapi

import (
	"net/http"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// AuditHandler 审计日志查询接口。
type AuditHandler struct {
	api *API
}

// NewAuditHandler 构造审计日志处理器。
func NewAuditHandler(api *API) *AuditHandler {
	return &AuditHandler{api: api}
}

// List GET /api/audit-logs 审计日志列表（按时间倒序，支持 limit/offset）。
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	logs := h.api.svc.Audit.List(0)
	OK(w, r, map[string]any{
		"logs":   logs,
		"total":  len(logs),
		"limit":  p.Limit,
		"offset": p.Offset,
	})
}
