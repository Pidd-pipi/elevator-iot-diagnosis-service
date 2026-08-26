package service

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// 并发读电梯 + 状态上报：读接口不得与采集写入共享内部可变对象。
func TestBug001ConcurrentGetIngestRace(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.ReportPeriod = 5
	cfg.EntrapmentThreshold = 30
	st := store.NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewIngestService(st, cfg, logger)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			if e, ok := st.Elevators.Get("ELEV-001"); ok {
				_ = e.HealthScore
				_ = e.Status
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			r := &domain.StateReport{
				ElevatorID: "ELEV-001", Floor: 3, Position: "3F",
				Direction: domain.DirectionIdle, Door: domain.DoorClosed,
				Leveling: true, Passenger: domain.PassengerNone,
				ReportedAt: time.Now(),
			}
			_, _ = svc.Ingest(r)
		}
	}()
	close(start)
	wg.Wait()
}

// 评分接口调用后，电梯台账必须持久化刷新后的评分。
func TestBug001ScorePersistedAfterGetScore(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	st.Faults.Append(&domain.FaultCodeLog{
		ElevatorID: "ELEV-001", FaultCode: "E01", Known: true, FaultType: domain.FaultKnown,
		OccurredAt: time.Now().Add(-time.Hour),
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewScoringService(st, cfg, logger)
	if _, err := svc.GetScore("ELEV-001"); err != nil {
		t.Fatal(err)
	}
	e, ok := st.Elevators.Get("ELEV-001")
	if !ok {
		t.Fatal("电梯缺失")
	}
	want := 100 - cfg.FaultScoreWeight
	if e.HealthScore != want {
		t.Fatalf("评分未持久化到台账: score=%d want %d", e.HealthScore, want)
	}
}
