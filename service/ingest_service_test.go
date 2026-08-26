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

func newTestIngest(t *testing.T) (*IngestService, *store.Store, *config.Config) {
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

func trapReport(elevatorID string, at time.Time) *domain.StateReport {
	return &domain.StateReport{
		ElevatorID:  elevatorID,
		Floor:       3,
		Position:    "3F-4F 之间",
		Direction:   domain.DirectionIdle,
		Door:        domain.DoorClosed,
		Leveling:    false,
		FaultCode:   "",
		Passenger:   domain.PassengerAlarm,
		AlarmActive: true,
		ReportedAt:  at,
	}
}

func TestIngestRejectsRunningWithOpenDoor(t *testing.T) {
	svc, st, _ := newTestIngest(t)
	at := time.Now()
	r := trapReport("ELEV-001", at)
	r.Direction = domain.DirectionUp
	r.Door = domain.DoorOpen
	if _, err := svc.Ingest(r); err == nil {
		t.Fatal("运行中开门的上报应被拒绝")
	}
	if len(st.Reports.All()) != 0 {
		t.Fatal("非法上报不应落库")
	}
}

func TestIngestUnknownFaultLogged(t *testing.T) {
	svc, st, _ := newTestIngest(t)
	at := time.Now()
	r := trapReport("ELEV-001", at)
	r.Leveling = true
	r.Floor = 3
	r.Passenger = domain.PassengerNone
	r.FaultCode = "X99"
	result, err := svc.Ingest(r)
	if err != nil {
		t.Fatalf("上报失败: %v", err)
	}
	if result.Diagnosis == nil || result.Diagnosis.Known {
		t.Fatal("未知故障码应被登记且 Known=false")
	}
	if st.Faults.CountUnknown() != 1 {
		t.Fatalf("未知故障记录数应为 1，得到 %d", st.Faults.CountUnknown())
	}
}

func TestIngestEntrapmentTriggerAfterThreshold(t *testing.T) {
	svc, _, cfg := newTestIngest(t)
	start := time.Now().Add(-2 * time.Minute)
	// 每 5 秒一条上报，累计 40 秒（8 条）超过 30 秒阈值。
	for i := 0; i < 8; i++ {
		at := start.Add(time.Duration(i) * cfg.ReportPeriod)
		result, err := svc.Ingest(trapReport("ELEV-001", at))
		if err != nil {
			t.Fatalf("第 %d 条上报失败: %v", i, err)
		}
		if result.EntrapmentState == "triggered" {
			if result.EntrapmentEvent == nil {
				t.Fatal("触发困人但事件为空")
			}
			return // 在第 8 条触发，符合预期
		}
	}
	t.Fatal("连续 40 秒困人条件应触发困人事件")
}

func TestIngestEntrapmentNoDuplicateEvent(t *testing.T) {
	svc, st, cfg := newTestIngest(t)
	start := time.Now().Add(-2 * time.Minute)
	var triggered *domain.EntrapmentEvent
	for i := 0; i < 8; i++ {
		at := start.Add(time.Duration(i) * cfg.ReportPeriod)
		result, err := svc.Ingest(trapReport("ELEV-001", at))
		if err != nil {
			t.Fatal(err)
		}
		if result.EntrapmentEvent != nil {
			triggered = result.EntrapmentEvent
		}
	}
	if triggered == nil {
		t.Fatal("应已触发困人事件")
	}
	// 继续上报，不应重复生成事件。
	for i := 8; i < 12; i++ {
		at := start.Add(time.Duration(i) * cfg.ReportPeriod)
		result, err := svc.Ingest(trapReport("ELEV-001", at))
		if err != nil {
			t.Fatal(err)
		}
		if result.EntrapmentEvent != nil {
			t.Fatal("未关闭事件存在时不应重复生成困人事件")
		}
	}
	open := st.Events.ListOpen()
	if len(open) != 1 {
		t.Fatalf("开放事件数应为 1，得到 %d", len(open))
	}
}

func TestIngestConditionBreakResets(t *testing.T) {
	svc, st, cfg := newTestIngest(t)
	start := time.Now().Add(-2 * time.Minute)
	for i := 0; i < 4; i++ {
		at := start.Add(time.Duration(i) * cfg.ReportPeriod)
		if _, err := svc.Ingest(trapReport("ELEV-001", at)); err != nil {
			t.Fatal(err)
		}
	}
	// 条件中断：平层 + 无乘客信号。
	r := trapReport("ELEV-001", start.Add(4*cfg.ReportPeriod))
	r.Leveling = true
	r.Floor = 5
	r.Passenger = domain.PassengerNone
	if _, err := svc.Ingest(r); err != nil {
		t.Fatal(err)
	}
	if st.Observations.IsActive("ELEV-001") {
		t.Fatal("条件中断后观测应被重置")
	}
}

func TestIngestElevatorNotFound(t *testing.T) {
	svc, _, _ := newTestIngest(t)
	_, err := svc.Ingest(trapReport("ELEV-999", time.Now()))
	if err == nil {
		t.Fatal("不存在的电梯上报应报错")
	}
}

func TestIngestBrokenChainDoesNotCarryStaleDuration(t *testing.T) {
	svc, _, cfg := newTestIngest(t)
	start := time.Now().Add(-2 * time.Minute)
	// 先建立 3 条连续观测（累计 10 秒）。
	for i := 0; i < 3; i++ {
		at := start.Add(time.Duration(i) * cfg.ReportPeriod)
		if _, err := svc.Ingest(trapReport("ELEV-001", at)); err != nil {
			t.Fatal(err)
		}
	}
	// 中断 2 小时后重新上报：不应携带旧时长瞬间触发。
	at := start.Add(2*time.Hour + cfg.ReportPeriod)
	result, err := svc.Ingest(trapReport("ELEV-001", at))
	if err != nil {
		t.Fatal(err)
	}
	if result.EntrapmentState == "triggered" {
		t.Fatal("中断超过两个周期后不应瞬间触发困人事件")
	}
	if result.ConsecutiveSeconds > int(cfg.ReportPeriod.Seconds()) {
		t.Fatalf("中断后应重新计时，累计时长不应超过一个周期，得到 %d", result.ConsecutiveSeconds)
	}
}
