package httpapi

import "net/http"

// ScoreHandler 健康评分接口。
type ScoreHandler struct {
	api *API
}

// NewScoreHandler 构造评分处理器。
func NewScoreHandler(api *API) *ScoreHandler {
	return &ScoreHandler{api: api}
}

// Score GET /api/elevators/{id}/score 健康评分明细。
func (h *ScoreHandler) Score(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := h.api.svc.Scoring.GetScore(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, map[string]any{"score": detail})
}
