package httpapi

import (
	"net/http"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/service"
)

// ElevatorHandler 电梯台账与列表接口。
type ElevatorHandler struct {
	svc *service.Services
}

// NewElevatorHandler 构造电梯处理器。
func NewElevatorHandler(svc *service.Services) *ElevatorHandler {
	return &ElevatorHandler{svc: svc}
}

// List GET /api/elevators 电梯列表（支持 ?q= 关键字过滤 + limit/offset 分页）。
func (h *ElevatorHandler) List(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	q := r.URL.Query().Get("q")
	elevators := h.svc.Store.Elevators.List()
	if q != "" {
		filtered := elevators[:0]
		for _, e := range elevators {
			if containsFold(e.ID, q) || containsFold(e.Building, q) || containsFold(e.Model, q) {
				filtered = append(filtered, e)
			}
		}
		elevators = filtered
	}
	page, total := paginate(elevators, p)
	OK(w, r, map[string]any{
		"elevators": page,
		"total":     total,
		"limit":     p.Limit,
		"offset":    p.Offset,
	})
}

// Get GET /api/elevators/{id} 电梯详情（含最近上报与当前评分）。
func (h *ElevatorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, ok := h.svc.Store.Elevators.Get(id)
	if !ok {
		Fail(w, r, wrapNotFound("电梯", id))
		return
	}
	latest, _ := h.svc.Store.Reports.LatestByElevator(id)
	score, err := h.svc.Scoring.GetScore(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, map[string]any{
		"elevator":      e,
		"latest_report": latest,
		"score":         score,
	})
}

// Watchlist GET /api/watchlist 重点关注名单。
func (h *ElevatorHandler) Watchlist(w http.ResponseWriter, r *http.Request) {
	OK(w, r, map[string]any{"elevators": h.svc.Store.Elevators.ListWatchlisted()})
}

// containsFold 忽略大小写判断子串。
func containsFold(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	sLower := toLower(s)
	subLower := toLower(sub)
	for i := 0; i+len(subLower) <= len(sLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

// toLower 简易 ASCII 小写转换（业务 ID/楼栋为 ASCII 即可）。
func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// wrapNotFound 构造统一的资源不存在错误。
func wrapNotFound(kind, id string) error {
	return &notFoundError{kind: kind, id: id}
}

// notFoundError 资源不存在错误。
type notFoundError struct {
	kind string
	id   string
}

// Error 实现 error 接口。
func (e *notFoundError) Error() string {
	return domain.ErrNotFound.Error() + ": " + e.kind + " " + e.id
}

// Unwrap 支持 errors.Is 匹配 ErrNotFound。
func (e *notFoundError) Unwrap() error {
	return domain.ErrNotFound
}
