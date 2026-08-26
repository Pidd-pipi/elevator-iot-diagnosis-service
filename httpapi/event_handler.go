package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/service"
	"example.com/elevator-iot-diagnosis-service/store"
)

// EventHandler 困人事件与处置流转接口。
type EventHandler struct {
	svc *service.Services
}

// NewEventHandler 构造事件处理器。
func NewEventHandler(svc *service.Services) *EventHandler {
	return &EventHandler{svc: svc}
}

// List GET /api/events 事件列表。
//
// 支持查询参数：elevator_id、status、limit、offset。
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	q := r.URL.Query()
	filter := store.EventFilter{ElevatorID: q.Get("elevator_id")}
	if raw := q.Get("status"); raw != "" {
		st, err := domain.ParseEventStatus(raw)
		if err != nil {
			Fail(w, r, domain.NewValidationError("status", err.Error()))
			return
		}
		filter.Status = st
	}
	events := h.svc.Events.List(filter)
	page, total := paginate(events, p)
	OK(w, r, map[string]any{
		"events": page,
		"total":  total,
		"limit":  p.Limit,
		"offset": p.Offset,
	})
}

// Get GET /api/events/{id} 事件详情（含处置任务与审计轨迹）。
func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	event, err := h.svc.Events.Get(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	disposal, _ := h.svc.Store.Disposals.GetByEvent(id)
	trail := h.svc.Audit.ListByEvent(id)
	OK(w, r, map[string]any{
		"event":    event,
		"disposal": disposal,
		"trail":    trail,
	})
}

// actorFromRequest 从请求头或查询参数提取操作者。
func actorFromRequest(r *http.Request) string {
	if a := r.Header.Get("X-Actor"); a != "" {
		return a
	}
	if a := r.URL.Query().Get("actor"); a != "" {
		return a
	}
	return "console"
}

// Accept POST /api/events/{id}/accept 接单处置。
func (h *EventHandler) Accept(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	event, err := h.svc.Events.Accept(id, actorFromRequest(r), time.Now())
	if err != nil {
		Fail(w, r, fmt.Errorf("接单处理失败: %w", err))
		return
	}
	OK(w, r, map[string]any{"event": event, "message": "接单成功，开始处置时限倒计时"})
}

// resolveRequest 处置完成请求体。
type resolveRequest struct {
	Disposer     string `json:"disposer"`
	Measure      string `json:"measure"`
	Note         string `json:"note"`
	RecoveryTime string `json:"recovery_time"`
}

// Resolve POST /api/events/{id}/resolve 处置完成（校验必填字段）。
func (h *EventHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req resolveRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Fail(w, r, domain.NewValidationError("body", "请求体 JSON 解析失败: "+err.Error()))
		return
	}
	event, err := h.svc.Events.Resolve(id, actorFromRequest(r), service.ResolveRequest{
		Disposer:     req.Disposer,
		Measure:      req.Measure,
		Note:         req.Note,
		RecoveryTime: req.RecoveryTime,
	}, time.Now())
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, map[string]any{"event": event, "message": "处置完成，事件已解除"})
}

// escalateRequest 升级请求体。
type escalateRequest struct {
	Reason string `json:"reason"`
}

// Escalate POST /api/events/{id}/escalate 升级处置（二次告警）。
// 请求体可选；缺省使用人工升级原因。
func (h *EventHandler) Escalate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req escalateRequest
	if r.Body != nil {
		if err := decodeJSON(w, r, &req); err != nil {
			Fail(w, r, domain.NewValidationError("body", "请求体 JSON 解析失败: "+err.Error()))
			return
		}
	}
	if req.Reason == "" {
		req.Reason = "现场情况恶化，人工申请升级"
	}
	event, err := h.svc.Events.Escalate(id, actorFromRequest(r), req.Reason, time.Now())
	if err != nil {
		Fail(w, r, fmt.Errorf("升级失败: %w", err))
		return
	}
	OK(w, r, map[string]any{"event": event, "message": "事件已升级并发送二次告警"})
}
