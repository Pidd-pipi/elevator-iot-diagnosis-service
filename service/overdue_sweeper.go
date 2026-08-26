package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/store"
)

// OverdueSweeper 困人超时扫描任务。
//
// 规则 3：接单后超过 10 分钟未处理（未开始处置/未解除）的事件，
// 由本扫描任务自动升级并二次告警。扫描周期默认 30 秒，可通过
// SWEEP_INTERVAL_SEC 调整。
type OverdueSweeper struct {
	store   *store.Store
	cfg     *config.Config
	logger  *slog.Logger
	events  *EventService
	onSweep func() // 每次扫描完成后的回调（用于周期性落盘）
}

// NewOverdueSweeper 构造超时扫描任务。
func NewOverdueSweeper(st *store.Store, cfg *config.Config, logger *slog.Logger, events *EventService) *OverdueSweeper {
	return &OverdueSweeper{store: st, cfg: cfg, logger: logger, events: events}
}

// SetOnSweep 注册扫描完成回调。
func (s *OverdueSweeper) SetOnSweep(fn func()) {
	s.onSweep = fn
}

// Run 启动定时扫描，直到 ctx 被取消。
func (s *OverdueSweeper) Run(ctx context.Context) {
	s.logger.Info("overdue sweeper started", "interval", s.cfg.SweepInterval)
	ticker := time.NewTicker(s.cfg.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.Sweep(now)
			if s.onSweep != nil {
				s.onSweep()
			}
		}
	}
}

// Sweep 执行一次超时扫描：将「已接单且超过处置时限」的开放事件自动升级。
//
// 扫描与升级都在当前 goroutine 内同步完成，返回本次升级的事件 ID 列表。
func (s *OverdueSweeper) Sweep(now time.Time) []string {
	var escalated []string
	for _, e := range s.store.Events.ListOpen() {
		if !e.IsOverdue(s.cfg.AcceptDeadline, now) {
			continue
		}
		reason := fmt.Sprintf("接单后超过 %s 未处理，自动升级", s.cfg.AcceptDeadline)
		if _, err := s.events.AutoEscalate(e.ID, reason, now); err != nil {
			s.logger.Error("auto escalate failed", "event", e.ID, "err", err)
			continue
		}
		escalated = append(escalated, e.ID)
		s.logger.Warn("event auto escalated", "event", e.ID, "elevator", e.ElevatorID)
	}
	return escalated
}

// SweepOnce 供测试直接触发一次扫描。
func (s *OverdueSweeper) SweepOnce(now time.Time) []string {
	return s.Sweep(now)
}
