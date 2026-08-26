package store

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

func countOpenFDs(t *testing.T) int {
	t.Helper()
	d, err := os.Open("/dev/fd")
	if err != nil {
		t.Skipf("无法读取 /dev/fd: %v", err)
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		t.Fatalf("读取 /dev/fd 失败: %v", err)
	}
	return len(names)
}

// 落盘失败（父路径非法）时 Save 必须返回错误，不能被 defer 吞掉。
func TestBug006SaveReportsWriteFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "afile")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blocker, "state.json")
	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	if err := st.Save(path); err == nil {
		t.Fatal("落盘失败时 Save 应返回错误，实际返回 nil（错误被吞掉）")
	}
}

// 连续成功落盘不得累积文件句柄（syncDir 不得泄漏目录句柄）。
func TestBug006SaveDoesNotLeakFDs(t *testing.T) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	base := countOpenFDs(t)
	for i := 0; i < 150; i++ {
		if err := st.Save(path); err != nil {
			t.Fatalf("保存快照失败: %v", err)
		}
	}
	after := countOpenFDs(t)
	if after-base > 50 {
		t.Fatalf("Save 泄漏文件句柄: %d -> %d", base, after)
	}
}

// 落盘失败时不得残留临时文件（错误分支漏清理）。
func TestBug006FailedSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	// 目标路径是已存在目录 → rename 必然失败。
	path := filepath.Join(dir, "target")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	if err := st.Save(path); err == nil {
		t.Fatal("rename 失败时 Save 应返回错误")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			t.Fatalf("落盘失败后残留临时文件: %s", e.Name())
		}
	}
}

// 损坏文件备份后原文件必须移走，再次加载应返回文件不存在。
func TestBug006CorruptBackupRemovesOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := NewStore()
	if err := st.Load(path); err == nil {
		t.Fatal("加载损坏文件应返回错误")
	}
	st2 := NewStore()
	err := st2.Load(path)
	if !os.IsNotExist(err) {
		t.Fatalf("备份后原文件应被移走，再次加载应报文件不存在，得到 %v", err)
	}
}

// 加载快照后全部子仓储必须完整恢复（含困人事件）。
func TestBug006LoadRestoresAllSubstores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	now := time.Now()
	ev := domain.NewEntrapmentEvent("event-1", "ELEV-001", now.Add(-40*time.Second), now, 40, "report-1")
	st.Events.Save(ev)
	if err := st.Save(path); err != nil {
		t.Fatal(err)
	}
	st2 := NewStore()
	if err := st2.Load(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := st2.Events.Get("event-1"); !ok {
		t.Fatal("加载后困人事件未恢复（Restore 漏恢复子仓储）")
	}
}
