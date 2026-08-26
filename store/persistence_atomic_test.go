package store

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/elevator-iot-diagnosis-service/domain"
)

func TestStoreLoadCorruptFileBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	if err := st.Save(path); err != nil {
		t.Fatalf("保存快照失败: %v", err)
	}
	// 写入损坏 JSON。
	if err := os.WriteFile(path, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	st2 := NewStore()
	if err := st2.Load(path); err == nil {
		t.Fatal("加载损坏文件应返回错误")
	}
	if st2.Elevators.Count() != 0 {
		t.Fatalf("加载失败后应降级为空库，得到电梯数 %d", st2.Elevators.Count())
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("损坏文件应备份为 .bak，得到 %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("损坏原文件应被移走，得到 %v", err)
	}
}

func TestStoreSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	st := NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	if err := st.Save(path); err != nil {
		t.Fatalf("保存快照失败: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || (len(e.Name()) > 0 && e.Name()[0] == '.') {
			t.Fatalf("目录不应残留临时文件: %s", e.Name())
		}
	}
}
