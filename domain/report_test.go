package domain

import (
	"testing"
	"time"
)

func baseReport() *StateReport {
	return &StateReport{
		ID:         "report-1",
		ElevatorID: "ELEV-001",
		Floor:      5,
		Position:   "5F",
		Direction:  DirectionIdle,
		Door:       DoorClosed,
		Leveling:   true,
		Passenger:  PassengerNone,
		ReportedAt: time.Now(),
	}
}

func TestReportValidateRunningWithOpenDoor(t *testing.T) {
	r := baseReport()
	r.Direction = DirectionUp
	r.Door = DoorOpen
	if err := r.ValidateState(18); err == nil {
		t.Fatal("运行中开门应被判定为非法状态")
	}
}

func TestReportValidateLevelingFloorRange(t *testing.T) {
	r := baseReport()
	r.Leveling = true
	r.Floor = 0
	if err := r.ValidateState(18); err == nil {
		t.Fatal("平层状态下楼层越界应被拒绝")
	}
	r.Floor = 19
	if err := r.ValidateState(18); err == nil {
		t.Fatal("平层状态下楼层超出服务楼层应被拒绝")
	}
}

func TestReportValidateNormal(t *testing.T) {
	r := baseReport()
	if err := r.ValidateState(18); err != nil {
		t.Fatalf("正常上报应通过校验: %v", err)
	}
	// 非平层 + 运行时楼层越界是允许的（井道内）。
	r.Leveling = false
	r.Floor = 19
	r.Direction = DirectionUp
	r.Door = DoorClosed
	if err := r.ValidateState(18); err != nil {
		t.Fatalf("非平层运行时楼层越界应允许: %v", err)
	}
}

func TestEntrapmentCondition(t *testing.T) {
	r := baseReport()
	r.Leveling = false
	r.Door = DoorClosed
	r.Passenger = PassengerAlarm
	if !r.EntrapmentCondition() {
		t.Fatal("非平层+关门+警铃 应满足困人条件")
	}
	r.Passenger = PassengerNone
	if r.EntrapmentCondition() {
		t.Fatal("无乘客信号不应满足困人条件")
	}
	r.Passenger = PassengerInfrared
	r.Leveling = true
	if r.EntrapmentCondition() {
		t.Fatal("平层状态下不应满足困人条件")
	}
}

func TestParseEnums(t *testing.T) {
	if d, err := ParseDirection("up"); err != nil || d != DirectionUp {
		t.Fatalf("解析方向失败: %v %v", d, err)
	}
	if d, err := ParseDoorStatus("closed"); err != nil || d != DoorClosed {
		t.Fatalf("解析门状态失败: %v %v", d, err)
	}
	if p, err := ParsePassengerSignal("both"); err != nil || p != PassengerBoth {
		t.Fatalf("解析乘客信号失败: %v %v", p, err)
	}
	if !DirectionUp.Moving() || DirectionIdle.Moving() {
		t.Fatal("Moving() 判定错误")
	}
}
