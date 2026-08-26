package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

func newBug009Ingest(t *testing.T) (*IngestService, *store.Store, *config.Config) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.ReportPeriod = 5 * time.Second
	cfg.EntrapmentThreshold = 30 * time.Second
	st := store.NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewIngestService(st, cfg, logger), st, cfg
}

func bug009TrapReport(at time.Time) *domain.StateReport {
	return &domain.StateReport{
		ElevatorID: "ELEV-001", Floor: 3, Position: "3F-4F 之间",
		Direction: domain.DirectionIdle, Door: domain.DoorClosed,
		Leveling: false, Passenger: domain.PassengerAlarm, AlarmActive: true,
		ReportedAt: at,
	}
}

// 困人条件中断后，观测必须被清空（不能复用陈旧观测）。
func TestBug009ConditionBreakResetsObservation(t *testing.T) {
	svc, st, cfg := newBug009Ingest(t)
	base := time.Now()
	r1 := bug009TrapReport(base)
	_, _ = svc.Ingest(r1)
	if !st.Observations.IsActive("ELEV-001") {
		t.Fatal("上报困人条件后应进入观测")
	}
	// 条件中断：门开、无乘客。
	r2 := &domain.StateReport{
		ElevatorID: "ELEV-001", Floor: 3, Position: "3F", Direction: domain.DirectionIdle,
		Door: domain.DoorOpen, Leveling: true, Passenger: domain.PassengerNone,
		ReportedAt: base.Add(cfg.ReportPeriod),
	}
	_, _ = svc.Ingest(r2)
	if st.Observations.IsActive("ELEV-001") {
		t.Fatal("条件中断后观测未清空，陈旧观测会被复用")
	}
}

// 观测期上报 ID 必须封顶，不能无限增长（内存泄漏）。
func TestBug009ReportIDsTrimmed(t *testing.T) {
	svc, st, cfg := newBug009Ingest(t)
	base := time.Now()
	for i := 0; i < 120; i++ {
		r := bug009TrapReport(base.Add(time.Duration(i) * cfg.ReportPeriod))
		_, _ = svc.Ingest(r)
	}
	obs, ok := st.Observations.Get("ELEV-001")
	if !ok {
		t.Fatal("观测记录缺失")
	}
	if len(obs.ReportIDs) > 50 {
		t.Fatalf("观测上报 ID 应封顶在 50 条，实际 %d（内存泄漏）", len(obs.ReportIDs))
	}
}

// 条件中断后 Elapsed 必须归零。
func TestBug009ElapsedResetsAfterBreak(t *testing.T) {
	_, st, _ := newBug009Ingest(t)
	base := time.Now()
	obs := &store.EntrapmentObservation{
		ElevatorID: "ELEV-001", Active: true, FirstSeenAt: base, LastReportAt: base,
		ConsecutiveSeconds: 300,
	}
	st.Observations.Set(obs)
	st.Observations.Reset("ELEV-001")
	if got := st.Observations.Elapsed("ELEV-001"); got != 0 {
		t.Fatalf("Reset 后 Elapsed 应归零，得到 %d", got)
	}
}

// 事件触发后重新计时，解除后再次上报不应立即重复触发。
func TestBug009NoImmediateRetriggerAfterResolve(t *testing.T) {
	svc, st, cfg := newBug009Ingest(t)
	base := time.Now()
	// 持续上报到触发事件。
	var last *domain.StateReport
	for i := 0; i <= 7; i++ {
		last = bug009TrapReport(base.Add(time.Duration(i) * cfg.ReportPeriod))
		_, _ = svc.Ingest(last)
	}
	if st.Events.CountByStatus(domain.EventAlerted) != 1 {
		t.Fatal("应触发一次困人事件")
	}
	// 解除事件。
	openEvents := st.Events.ListOpen()
	if len(openEvents) != 1 {
		t.Fatalf("应有一个开放事件，得到 %d", len(openEvents))
	}
	ev := openEvents[0]
	_ = ev.Accept(last.ReportedAt)
	_ = ev.Release(last.ReportedAt.Add(cfg.ReportPeriod))
	st.Events.Save(ev)
	// 条件中断（紧跟触发后一个周期）。
	brk := &domain.StateReport{
		ElevatorID: "ELEV-001", Floor: 3, Position: "3F", Direction: domain.DirectionIdle,
		Door: domain.DoorOpen, Leveling: true, Passenger: domain.PassengerNone,
		ReportedAt: last.ReportedAt.Add(cfg.ReportPeriod),
	}
	_, _ = svc.Ingest(brk)
	// 仅一条困人条件上报（中断后一个周期），不应立即重新触发（累计需超过阈值）。
	one := bug009TrapReport(last.ReportedAt.Add(2 * cfg.ReportPeriod))
	_, _ = svc.Ingest(one)
	if got := len(st.Events.List(store.EventFilter{})); got != 1 {
		t.Fatalf("单条上报不应立即重新触发困人事件，事件总数 %d（应保持 1）", got)
	}
}
