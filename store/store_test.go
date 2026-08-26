package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

func TestStorePersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	now := time.Now()
	ev := domain.NewEntrapmentEvent("event-1", "ELEV-001", now.Add(-40*time.Second), now, 40, "report-1")
	st.Events.Save(ev)
	st.Audits.Append(domain.NewAuditLog("event.alert", "system", "event", "event-1", "告警", now))

	if err := st.Save(path); err != nil {
		t.Fatalf("保存快照失败: %v", err)
	}

	st2 := NewStore()
	if err := st2.Load(path); err != nil {
		t.Fatalf("加载快照失败: %v", err)
	}
	if st2.Elevators.Count() != 1 {
		t.Fatalf("电梯数应为 1，得到 %d", st2.Elevators.Count())
	}
	e, ok := st2.Elevators.Get("ELEV-001")
	if !ok || e.Building != "A 栋" {
		t.Fatal("电梯数据恢复不完整")
	}
	ev2, ok := st2.Events.Get("event-1")
	if !ok || ev2.Status != domain.EventAlerted {
		t.Fatal("事件数据恢复不完整")
	}
	if st2.Audits.Count() != 1 {
		t.Fatalf("审计日志数应为 1，得到 %d", st2.Audits.Count())
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	st := NewStore()
	err := st.Load(filepath.Join(t.TempDir(), "none.json"))
	if err == nil {
		t.Fatal("加载不存在的文件应返回错误")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("应返回文件不存在错误，得到 %v", err)
	}
}

func TestStoreIDUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID("event")
		if seen[id] {
			t.Fatalf("ID 重复: %s", id)
		}
		seen[id] = true
	}
}
