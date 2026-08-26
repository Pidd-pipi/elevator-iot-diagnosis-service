package httpapi

import (
	"net/http"
	"sort"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// DiagnoseHandler 故障诊断接口。
type DiagnoseHandler struct {
	api *API
}

// NewDiagnoseHandler 构造诊断处理器。
func NewDiagnoseHandler(api *API) *DiagnoseHandler {
	return &DiagnoseHandler{api: api}
}

// Rules GET /api/diagnosis 故障码诊断映射表 + 未知故障记录（未知记录支持分页）。
func (h *DiagnoseHandler) Rules(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	unknown := h.api.svc.Diagnose.ListUnknown(0)
	unknownPage, unknownTotal := paginate(unknown, p)
	rules := h.api.svc.Diagnose.Rules()
	sort.Slice(rules, func(i, j int) bool { return rules[i].Code < rules[j].Code })
	OK(w, r, map[string]any{
		"rules":         rules,
		"unknown":       unknownPage,
		"unknown_cnt":   h.api.svc.Diagnose.UnknownCount(),
		"unknown_total": unknownTotal,
		"limit":         p.Limit,
		"offset":        p.Offset,
	})
}

// Faults GET /api/elevators/{id}/faults 故障码时间线（支持 limit/offset 分页）。
func (h *DiagnoseHandler) Faults(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// 电梯不存在时直接返回 404，不得以 200 返回空列表。
	if _, ok := h.api.svc.Store.Elevators.Get(id); !ok {
		Fail(w, r, wrapNotFound("电梯", id))
		return
	}
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	logs := h.api.svc.Diagnose.ListByElevator(id, 0)
	page, total := paginate(logs, p)
	OK(w, r, map[string]any{
		"faults": page,
		"total":  total,
		"limit":  p.Limit,
		"offset": p.Offset,
	})
}
