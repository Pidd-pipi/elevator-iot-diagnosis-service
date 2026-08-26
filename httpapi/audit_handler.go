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
//
// 修复前：硬编码 List(0) 后直接返回全部，忽略 limit/offset，导致
// 条数限制不生效。此处改为与其他列表接口一致的口径：取全量倒序切片，
// 再用 paginate 按 limit/offset 截取当前页，并返回 total。
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	logs := h.api.svc.Audit.List(0)
	page, total := paginate(logs, p)
	OK(w, r, map[string]any{
		"logs":   page,
		"total":  total,
		"limit":  p.Limit,
		"offset": p.Offset,
	})
}
