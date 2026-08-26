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

func TestSweeperEscalatesOverdueAcceptedEvents(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.AcceptDeadline = 10 * time.Minute
	st := store.NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)

	now := time.Now()
	// 事件 1：11 分钟前接单 → 应被扫描升级。
	ev1 := domain.NewEntrapmentEvent("event-1", "ELEV-001", now.Add(-20*time.Minute), now.Add(-20*time.Minute), 40, "")
	_ = ev1.Accept(now.Add(-11 * time.Minute))
	st.Events.Save(ev1)
	// 事件 2：2 分钟前接单 → 不应升级。
	ev2 := domain.NewEntrapmentEvent("event-2", "ELEV-001", now.Add(-5*time.Minute), now.Add(-5*time.Minute), 40, "")
	_ = ev2.Accept(now.Add(-2 * time.Minute))
	st.Events.Save(ev2)

	escalated := svc.Sweeper.Sweep(now)
	if len(escalated) != 1 || escalated[0] != "event-1" {
		t.Fatalf("应仅升级 event-1，得到 %v", escalated)
	}
	got, _ := st.Events.Get("event-1")
	if got.Status != domain.EventEscalated || !got.SecondAlarmSent {
		t.Fatal("超时事件应被自动升级并二次告警")
	}
	got2, _ := st.Events.Get("event-2")
	if got2.Status != domain.EventAccepted {
		t.Fatalf("未超时事件不应被升级，得到 %s", got2.Status)
	}
	// 审计留痕。
	if st.Audits.Count() == 0 {
		t.Fatal("自动升级应写入审计日志")
	}
}

func TestSweeperSkipsTerminalEvents(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)

	now := time.Now()
	ev := domain.NewEntrapmentEvent("event-1", "ELEV-001", now.Add(-20*time.Minute), now.Add(-20*time.Minute), 40, "")
	_ = ev.Accept(now.Add(-11 * time.Minute))
	_ = ev.Escalate(now, "已升级")
	st.Events.Save(ev)

	escalated := svc.Sweeper.Sweep(now)
	if len(escalated) != 0 {
		t.Fatalf("终态事件不应被重复扫描升级，得到 %v", escalated)
	}
}
