package httpapi

import (
	"net/http"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// WatchlistHandler 重点关注名单接口。
type WatchlistHandler struct {
	api *API
}

// NewWatchlistHandler 构造重点关注处理器。
func NewWatchlistHandler(api *API) *WatchlistHandler {
	return &WatchlistHandler{api: api}
}

// Watchlist GET /api/watchlist 重点关注电梯列表（支持 limit/offset 分页）。
func (h *WatchlistHandler) Watchlist(w http.ResponseWriter, r *http.Request) {
	p, err := parsePagination(r)
	if err != nil {
		Fail(w, r, domain.NewValidationError("pagination", err.Error()))
		return
	}
	items := h.api.svc.Store.Elevators.ListWatchlisted()
	page, total := paginate(items, p)
	OK(w, r, map[string]any{
		"elevators": page,
		"total":     total,
		"limit":     p.Limit,
		"offset":    p.Offset,
	})
}
