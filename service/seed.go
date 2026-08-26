package service

import (
	"fmt"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// SeedService 负责在首次启动时写入演示基线数据，
// 保证各页面与接口开箱即可展示真实业务链路。
type SeedService struct {
	store  *store.Store
	cfg    *config.Config
	ingest *IngestService
}

// NewSeedService 构造种子服务。
func NewSeedService(st *store.Store, cfg *config.Config, ingest *IngestService) *SeedService {
	return &SeedService{store: st, cfg: cfg, ingest: ingest}
}

// EnsureSeed 在仓储为空时写入演示数据；已有数据时跳过。
func (s *SeedService) EnsureSeed() error {
	if s.store.Elevators.Count() > 0 {
		return nil
	}
	now := time.Now()
	s.seedElevators(now)
	s.seedHistory(now)
	return nil
}

// seedElevators 写入 6 台演示电梯台账。
func (s *SeedService) seedElevators(now time.Time) {
	elevators := []struct {
		id, building, model, install string
		capacity, floors             int
	}{
		{"ELEV-001", "A 栋", "OTIS-Gen2", "2019-05-12", 1000, 18},
		{"ELEV-002", "A 栋", "Mitsubishi-ELENESSA", "2020-08-01", 1000, 18},
		{"ELEV-003", "B 栋", "KONE-MonoSpace", "2018-11-20", 1350, 26},
		{"ELEV-004", "B 栋", "Schindler-5500", "2021-03-15", 1350, 26},
		{"ELEV-005", "C 栋", "Hitachi-DF3", "2017-06-30", 800, 12},
		{"ELEV-006", "C 栋", "ThyssenKrupp-5300", "2022-01-10", 1000, 15},
	}
	for _, e := range elevators {
		s.store.Elevators.Save(domain.NewElevator(e.id, e.building, e.model, e.install, e.capacity, e.floors))
	}
}

// seedHistory 写入历史上报、故障记录与一条已闭环/一条进行中的困人事件，
// 让评分与事件页有真实数据可展示。
func (s *SeedService) seedHistory(now time.Time) {
	// 历史故障记录：ELEV-005 多次故障 + 一次未按时处置 → 进入重点关注。
	faultHistory := []struct {
		elevator string
		code     string
		hoursAgo int
	}{
		{"ELEV-005", "E01", 5},
		{"ELEV-005", "E03", 28},
		{"ELEV-005", "E02", 50},
		{"ELEV-005", "E01", 80},
		{"ELEV-005", "X99", 100}, // 未知故障码
		{"ELEV-003", "E06", 30},
		{"ELEV-001", "E07", 10},
	}
	for _, f := range faultHistory {
		at := now.Add(-time.Duration(f.hoursAgo) * time.Hour)
		report := &domain.StateReport{
			ID:         store.NewID("report"),
			ElevatorID: f.elevator,
			Floor:      1,
			Position:   "1F",
			Direction:  domain.DirectionIdle,
			Door:       domain.DoorClosed,
			Leveling:   true,
			FaultCode:  f.code,
			ReportedAt: at,
			CreatedAt:  at,
		}
		s.store.Reports.Append(report)
		_ = s.ingest.diagnose(f.elevator, f.code, report.ID, at, s.cfg.UnknownFaultPrompt)
	}

	// 一条已闭环事件：ELEV-003，处置及时。
	s.seedClosedEvent(now)

	// 一条进行中事件：ELEV-001 已接单（模拟刚接单，未超时）。
	s.seedOpenEvent(now)

	// 全量刷新评分。
	_ = s.refreshAllScores(now)
}

// seedClosedEvent 构造 ELEV-003 已解除事件及其处置记录。
func (s *SeedService) seedClosedEvent(now time.Time) {
	started := now.Add(-3 * time.Hour)
	accepted := started.Add(2 * time.Minute)
	recovered := accepted.Add(8 * time.Minute)
	event := domain.NewEntrapmentEvent(store.NewID("event"), "ELEV-003", started, started, 35, "")
	if err := event.Accept(accepted); err != nil {
		panic(err)
	}
	if err := event.Release(recovered); err != nil {
		panic(err)
	}
	s.store.Events.Save(event)

	disposal := domain.NewDisposalRecord(store.NewID("disposal"), event.ID, "ELEV-003", accepted)
	_ = disposal.Complete("王工", "更换门锁触点并复位", "现场排查正常", recovered, s.cfg.AcceptDeadline)
	s.store.Disposals.Save(disposal)

	s.store.Audits.Append(domain.NewAuditLog("event.accept", "王工", "event", event.ID,
		fmt.Sprintf("电梯 ELEV-003 困人事件接单"), accepted))
	s.store.Audits.Append(domain.NewAuditLog("event.resolve", "王工", "event", event.ID,
		fmt.Sprintf("电梯 ELEV-003 困人解除"), recovered))
}

// seedOpenEvent 构造 ELEV-001 已接单、进行中的事件。
func (s *SeedService) seedOpenEvent(now time.Time) {
	started := now.Add(-12 * time.Minute)
	accepted := now.Add(-10 * time.Minute)
	event := domain.NewEntrapmentEvent(store.NewID("event"), "ELEV-001", started, started, 40, "")
	if err := event.Accept(accepted); err != nil {
		panic(err)
	}
	s.store.Events.Save(event)
	disposal := domain.NewDisposalRecord(store.NewID("disposal"), event.ID, "ELEV-001", accepted)
	s.store.Disposals.Save(disposal)
	s.store.Audits.Append(domain.NewAuditLog("event.accept", "李工", "event", event.ID,
		"电梯 ELEV-001 困人事件接单", accepted))
}

// refreshAllScores 刷新全部电梯评分。
func (s *SeedService) refreshAllScores(now time.Time) int {
	count := 0
	for _, e := range s.store.Elevators.List() {
		_, err := ComputeScoreFor(s.store, s.cfg, e.ID, now)
		if err != nil {
			continue
		}
		count++
	}
	return count
}
