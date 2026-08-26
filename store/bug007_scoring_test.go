package store

import (
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// 未知故障列表不得混入已知故障。
func TestBug007ListUnknownExcludesKnown(t *testing.T) {
	s := NewFaultStore()
	now := time.Now()
	s.Append(&domain.FaultCodeLog{
		ElevatorID: "ELEV-001", FaultCode: "E01", Diagnosis: "门锁回路", Known: true,
		FaultType: domain.FaultKnown, OccurredAt: now,
	})
	s.Append(&domain.FaultCodeLog{
		ElevatorID: "ELEV-001", FaultCode: "X99", Diagnosis: "未知", Known: false,
		FaultType: domain.FaultUnknown, OccurredAt: now,
	})
	unknown := s.ListUnknown(0)
	if len(unknown) != 1 {
		t.Fatalf("未知故障列表应只含未知故障，得到 %d 条（混入已知故障）", len(unknown))
	}
	if unknown[0].FaultCode != "X99" {
		t.Fatalf("未知列表应包含 X99，得到 %v", unknown[0].FaultCode)
	}
}
