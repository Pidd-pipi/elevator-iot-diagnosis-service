package domain

import "time"

// ScoreDetail 健康评分明细。
//
// 评分规则（规则 5）：
//
//	score = 100 - 近30天故障次数×系数 - 未按时处置次数×系数
//
// 评分 ≤ WatchlistThreshold（默认 60）的电梯进入重点关注名单。
type ScoreDetail struct {
	ElevatorID string `json:"elevator_id"`
	// Score 最终评分（0-100，下限 0）。
	Score int `json:"score"`
	// FaultCount 统计窗口内故障次数。
	FaultCount int `json:"fault_count"`
	// UntimelyCount 统计窗口内未按时处置次数。
	UntimelyCount int `json:"untimely_count"`
	// DeductionFault 故障扣分。
	DeductionFault int `json:"deduction_fault"`
	// DeductionUntimely 未按时处置扣分。
	DeductionUntimely int `json:"deduction_untimely"`
	// Watchlisted 是否进入重点关注名单。
	Watchlisted bool `json:"watchlisted"`
	// Since 统计窗口起点。
	Since time.Time `json:"since"`
	// Until 统计窗口终点。
	Until time.Time `json:"until"`
	// Reasons 扣分原因明细（便于前端展示）。
	Reasons []string `json:"reasons"`
}

// ComputeScore 依据故障次数与未按时处置次数计算健康评分。
func ComputeScore(faultCount, untimelyCount, faultWeight, untimelyWeight, watchlistThreshold int) (score int, watchlisted bool) {
	score = 100 - faultCount*faultWeight - untimelyCount*untimelyWeight
	if score < 0 {
		score = 0
	}
	watchlisted = score <= watchlistThreshold
	return score, watchlisted
}
