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

func newTestEventSvc(t *testing.T) (*EventService, *store.Store) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.AcceptDeadline = 10 * time.Minute
	st := store.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)
	return svc.Events, st
}

func makeOpenEvent(t *testing.T, st *store.Store, elevatorID string) *domain.EntrapmentEvent {
	t.Helper()
	now := time.Now()
	event := domain.NewEntrapmentEvent(store.NewID("event"), elevatorID, now.Add(-40*time.Second), now, 40, "")
	st.Events.Save(event)
	return event
}

func TestAcceptThenResolveFlow(t *testing.T) {
	svc, st := newTestEventSvc(t)
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	event := makeOpenEvent(t, st, "ELEV-001")

	now := time.Now()
	accepted, err := svc.Accept(event.ID, "王工", now)
	if err != nil {
		t.Fatalf("接单失败: %v", err)
	}
	if accepted.Status != domain.EventAccepted {
		t.Fatalf("接单后状态应为 accepted，得到 %s", accepted.Status)
	}
	disposal, ok := st.Disposals.GetByEvent(event.ID)
	if !ok {
		t.Fatal("接单后应创建处置任务")
	}
	if disposal.AcceptedAt.IsZero() {
		t.Fatal("处置任务应记录接单时间")
	}

	// 缺少必填字段的处置应被拒绝。
	_, err = svc.Resolve(event.ID, "王工", ResolveRequest{}, now.Add(5*time.Minute))
	if err == nil {
		t.Fatal("缺少处置字段应被拒绝")
	}

	// 完整处置。
	recovery := now.Add(8 * time.Minute)
	resolved, err := svc.Resolve(event.ID, "王工", ResolveRequest{
		Disposer:     "王工",
		Measure:      "更换门锁触点并复位",
		RecoveryTime: recovery.Format(time.RFC3339),
	}, recovery)
	if err != nil {
		t.Fatalf("处置完成失败: %v", err)
	}
	if resolved.Status != domain.EventReleased {
		t.Fatalf("处置后状态应为 released，得到 %s", resolved.Status)
	}
	disposal, _ = st.Disposals.GetByEvent(event.ID)
	if !disposal.Timely {
		t.Fatal("8 分钟内处置应判定按时")
	}

	// 已解除事件不可重复处置。
	if _, err := svc.Resolve(event.ID, "王工", ResolveRequest{
		Disposer:     "王工",
		Measure:      "x",
		RecoveryTime: recovery.Format(time.RFC3339),
	}, recovery.Add(time.Minute)); err == nil {
		t.Fatal("已闭环事件不可重复处置")
	}
}

func TestEscalateFlow(t *testing.T) {
	svc, st := newTestEventSvc(t)
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	event := makeOpenEvent(t, st, "ELEV-001")
	now := time.Now()
	escalated, err := svc.Escalate(event.ID, "李工", "现场情况恶化", now)
	if err != nil {
		t.Fatalf("升级失败: %v", err)
	}
	if escalated.Status != domain.EventEscalated || !escalated.SecondAlarmSent {
		t.Fatal("升级后应进入 escalated 并发送二次告警")
	}
	if escalated.EscalationCount != 1 {
		t.Fatalf("升级次数应为 1，得到 %d", escalated.EscalationCount)
	}
	if _, err := svc.Accept(event.ID, "李工", now); err == nil {
		t.Fatal("已升级事件不可接单")
	}
}

func TestAutoEscalate(t *testing.T) {
	svc, st := newTestEventSvc(t)
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	event := makeOpenEvent(t, st, "ELEV-001")
	now := time.Now()
	if _, err := svc.Accept(event.ID, "王工", now.Add(-11*time.Minute)); err != nil {
		t.Fatal(err)
	}
	escalated, err := svc.AutoEscalate(event.ID, "接单超时自动升级", now)
	if err != nil {
		t.Fatalf("自动升级失败: %v", err)
	}
	if escalated.Status != domain.EventEscalated {
		t.Fatalf("自动升级后状态应为 escalated，得到 %s", escalated.Status)
	}
}
