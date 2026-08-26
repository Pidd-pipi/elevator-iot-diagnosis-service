package service

import (
	"fmt"
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// ScoringService 健康评分服务。
//
// 规则 5：score = 100 - 近30天故障次数×系数 - 未按时处置次数×系数；
// 评分 ≤ WatchlistThreshold（默认 60）的电梯进入重点关注名单。
type ScoringService struct {
	store  *store.Store
	cfg    *config.Config
	logger *slog.Logger
}

// NewScoringService 构造评分服务。
func NewScoringService(st *store.Store, cfg *config.Config, logger *slog.Logger) *ScoringService {
	return &ScoringService{store: st, cfg: cfg, logger: logger}
}

// GetScore 计算指定电梯的健康评分明细，并写回电梯台账。
func (s *ScoringService) GetScore(elevatorID string) (*domain.ScoreDetail, error) {
	if _, ok := s.store.Elevators.Get(elevatorID); !ok {
		return nil, fmt.Errorf("%w: 电梯 %s", domain.ErrNotFound, elevatorID)
	}
	return ComputeScoreFor(s.store, s.cfg, elevatorID, time.Now())
}

// ComputeScoreFor 计算评分明细并同步电梯台账（供其他服务复用）。
func ComputeScoreFor(st *store.Store, cfg *config.Config, elevatorID string, now time.Time) (*domain.ScoreDetail, error) {
	_, ok := st.Elevators.Get(elevatorID)
	if !ok {
		return nil, fmt.Errorf("%w: 电梯 %s", domain.ErrNotFound, elevatorID)
	}
	detail := computeScoreDetail(st, cfg, elevatorID, now)
	return detail, nil
}

// computeScoreDetail 纯计算：从仓储统计窗口内故障次数与未按时处置次数。
func computeScoreDetail(st *store.Store, cfg *config.Config, elevatorID string, now time.Time) *domain.ScoreDetail {
	since := now.Add(-cfg.ScoreWindow)
	faultCount := st.Faults.CountByElevatorSince(elevatorID, since)
	untimelyCount := st.Disposals.CountUntimelyByElevatorSince(elevatorID, since)

	score, watchlisted := domain.ComputeScore(
		faultCount,
		untimelyCount,
		cfg.FaultScoreWeight,
		cfg.UntimelyScoreWeight,
		cfg.WatchlistThreshold,
	)

	detail := &domain.ScoreDetail{
		ElevatorID:        elevatorID,
		Score:             score,
		FaultCount:        faultCount,
		UntimelyCount:     untimelyCount,
		DeductionFault:    faultCount * cfg.FaultScoreWeight,
		DeductionUntimely: untimelyCount * cfg.UntimelyScoreWeight,
		Watchlisted:       watchlisted,
		Since:             since,
		Until:             now,
	}
	if faultCount > 0 {
		detail.Reasons = append(detail.Reasons,
			fmt.Sprintf("近30天发生 %d 次故障，每次扣 %d 分", faultCount, cfg.FaultScoreWeight))
	}
	if untimelyCount > 0 {
		detail.Reasons = append(detail.Reasons,
			fmt.Sprintf("近30天有 %d 次未按时处置，每次扣 %d 分", untimelyCount, cfg.UntimelyScoreWeight))
	}
	if len(detail.Reasons) == 0 {
		detail.Reasons = append(detail.Reasons, "近30天无故障、处置及时，保持满分")
	}
	if watchlisted {
		detail.Reasons = append(detail.Reasons,
			fmt.Sprintf("评分 ≤ %d，已进入重点关注名单", cfg.WatchlistThreshold))
	}
	return detail
}

// RefreshAll 刷新全部电梯的评分（启动与定时任务可调用）。
func (s *ScoringService) RefreshAll(now time.Time) int {
	count := 0
	for _, e := range s.store.Elevators.List() {
		detail, err := ComputeScoreFor(s.store, s.cfg, e.ID, now)
		if err != nil {
			s.logger.Warn("refresh score failed", "elevator", e.ID, "err", err)
			continue
		}
		_ = detail
		count++
	}
	return count
}
