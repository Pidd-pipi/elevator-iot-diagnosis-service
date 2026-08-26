package store

import (
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// 审计仓储从 nil 快照恢复后追加不得 panic（nil map 需初始化）。
func TestBug004AuditRestoreThenAppendNoPanic(t *testing.T) {
	s := NewAuditStore()
	s.Restore(nil)
	s.Append(domain.NewAuditLog("event.alert", "system", "event", "e1", "告警", time.Now()))
	if s.Count() != 1 {
		t.Fatalf("审计写入后应可读，得到 %d", s.Count())
	}
}

// 处置仓储从 nil 快照恢复后保存不得 panic。
func TestBug004DisposalRestoreThenSaveNoPanic(t *testing.T) {
	s := NewDisposalStore()
	s.Restore(nil)
	s.Save(domain.NewDisposalRecord("d1", "e1", "ELEV-001", time.Now()))
	if len(s.All()) != 1 {
		t.Fatalf("处置写入后应可读，得到 %d", len(s.All()))
	}
}
