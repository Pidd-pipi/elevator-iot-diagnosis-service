package httpapi

import (
	"net/http"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// OverviewHandler 总览聚合接口。
type OverviewHandler struct {
	api *API
}

// NewOverviewHandler 构造总览处理器。
func NewOverviewHandler(api *API) *OverviewHandler {
	return &OverviewHandler{api: api}
}

// Overview GET /api/overview 总览聚合数据。
//
// 供前端「/ 电梯总览」页面一次性拉取：统计卡片 + 进行中困人事件 +
// 最近上报 + 最近审计。
func (h *OverviewHandler) Overview(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	openEvents := h.api.svc.Store.Events.ListOpen()
	// 最多取 5 条进行中事件供总览展示。
	if len(openEvents) > 5 {
		openEvents = openEvents[:5]
	}
	recent := h.api.svc.Ingest.ListRecentReports(0)
	audits := h.api.svc.Audit.List(10)

	stats := map[string]any{
		"elevator_total":  h.api.svc.Store.Elevators.Count(),
		"watchlist_total": len(h.api.svc.Store.Elevators.ListWatchlisted()),
		"open_events":     h.api.svc.Store.Events.CountOpen(),
		"reports_today":   h.api.svc.Store.Reports.CountToday(now),
		"unknown_faults":  h.api.svc.Diagnose.UnknownCount(),
		"audit_total":     h.api.svc.Store.Audits.Count(),
		"released_total":  h.api.svc.Store.Events.CountByStatus(domain.EventReleased),
		"escalated_total": h.api.svc.Store.Events.CountByStatus(domain.EventEscalated),
		"timestamp":       now.Format(time.RFC3339),
	}
	OK(w, r, map[string]any{
		"stats":          stats,
		"open_events":    openEvents,
		"recent_reports": recent,
		"recent_audits":  audits,
	})
}
