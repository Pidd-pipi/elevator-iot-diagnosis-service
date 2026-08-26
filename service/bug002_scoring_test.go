package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

func newBug002Sweeper(t *testing.T) (*OverdueSweeper, *store.Store, *config.Config) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.SweepInterval = 10 * time.Millisecond
	st := store.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)
	return svc.Sweeper, st, cfg
}

func seedBug002OverdueEvent(t *testing.T, st *store.Store, elevatorID string, acceptedAgo time.Duration) string {
	t.Helper()
	now := time.Now()
	started := now.Add(-acceptedAgo - time.Hour)
	accepted := now.Add(-acceptedAgo)
	ev := domain.NewEntrapmentEvent(store.NewID("event"), elevatorID, started, started, 40, "")
	if err := ev.Accept(accepted); err != nil {
		t.Fatal(err)
	}
	st.Events.Save(ev)
	return ev.ID
}

// 取消后扫描任务必须及时退出，不残留 goroutine/定时器。
func TestBug002RunExitsOnCancel(t *testing.T) {
	sw, _, _ := newBug002Sweeper(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run 未在取消后退出，扫描任务 goroutine/定时器泄漏")
	}
}

// 单次扫描必须同步升级全部逾期事件。
func TestBug002SweepEscalatesAllOverdue(t *testing.T) {
	sw, st, _ := newBug002Sweeper(t)
	id1 := seedBug002OverdueEvent(t, st, "ELEV-001", 30*time.Minute)
	id2 := seedBug002OverdueEvent(t, st, "ELEV-002", 45*time.Minute)
	got := sw.Sweep(time.Now())
	if len(got) != 2 {
		t.Fatalf("扫描应升级 2 个逾期事件，得到 %d: %v", len(got), got)
	}
	for _, id := range []string{id1, id2} {
		e, _ := st.Events.Get(id)
		if e == nil || e.Status != domain.EventEscalated {
			t.Fatalf("事件 %s 未升级，状态 %v", id, e.Status)
		}
	}
}

// Run 每次扫描触发后 onSweep 回调必须已执行（周期性落盘不丢）。
func TestBug002OnSweepInvokedAfterSweep(t *testing.T) {
	sw, _, _ := newBug002Sweeper(t)
	called := make(chan struct{}, 10)
	sw.SetOnSweep(func() { called <- struct{}{} })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("Run 触发扫描后 onSweep 未执行，回调被泄漏的 goroutine 卡住")
	}
}
