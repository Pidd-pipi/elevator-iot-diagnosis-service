package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// Snapshot 全量数据快照，用于 JSON 文件持久化。
type Snapshot struct {
	Version      int                                `json:"version"`
	SavedAt      time.Time                          `json:"saved_at"`
	Elevators    map[string]*domain.Elevator        `json:"elevators"`
	Reports      map[string]*domain.StateReport     `json:"reports"`
	Events       map[string]*domain.EntrapmentEvent `json:"events"`
	Disposals    map[string]*domain.DisposalRecord  `json:"disposals"`
	FaultLogs    map[string]*domain.FaultCodeLog    `json:"fault_logs"`
	Audits       map[string]*domain.AuditLog        `json:"audits"`
	Observations map[string]*EntrapmentObservation  `json:"observations"`
}

// snapshotVersion 快照格式版本号。
const snapshotVersion = 1

// Save 将全量数据原子写入 JSON 文件。
//
// 原子写流程：同目录临时文件 → 写数据 → fsync 文件 → close →
// rename 原子替换 → fsync 目录，保证进程崩溃时不会留下半截文件。
//
// 任何错误（磁盘满、fsync 失败、rename 失败等）都原样上抛，由调用方记录
// 日志并决定降级策略；同时通过 defer 保证临时文件句柄被关闭、临时文件被
// 清理，避免 FD 泄漏与磁盘残留。
func (s *Store) Save(path string) error {
	if path == "" {
		return nil
	}
	s.persistMu.Lock()
	defer s.persistMu.Unlock()

	snap := s.Snapshot()
	snap.Version = snapshotVersion
	snap.SavedAt = time.Now()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建持久化目录失败: %w", err)
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化快照失败: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时快照失败: %w", err)
	}
	tmpName := tmp.Name()
	// 兜底清理：无论成功失败都关闭句柄并删除临时文件，避免 FD 泄漏与残留
	// 临时文件。成功路径下 rename 后 tmpName 已不存在，Close/Remove 退化为
	// 无害的 no-op（Close 对已关闭句柄返回 ErrClosed，Remove 返回 IsNotExist）。
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("设置临时快照权限失败: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("写入临时快照失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync 临时快照失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时快照失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("原子替换快照失败: %w", err)
	}
	syncDir(dir)
	return nil
}

// syncDir fsync 目录，确保 rename 结果持久化；目录打开/同步失败不阻断主流程，
// 仅作尽力而为的持久化保障（句柄始终关闭，避免 FD 泄漏）。
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	_ = d.Sync()
}

// Load 从 JSON 文件恢复全量数据；文件不存在时返回 os.ErrNotExist。
//
// 若文件损坏（JSON 解析失败），先将损坏文件备份为 <path>.bak，
// 再返回错误。调用方（main）据此告警并降级为空库启动。
func (s *Store) Load(path string) error {
	if path == "" {
		return nil
	}
	s.persistMu.Lock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		if bakErr := backupCorruptFile(path); bakErr != nil {
			return fmt.Errorf("解析快照失败: %w（且备份损坏文件失败: %v）", err, bakErr)
		}
		return fmt.Errorf("解析快照失败: %w（已备份损坏文件到 %s.bak）", err, path)
	}
	if snap.Version > snapshotVersion {
		if bakErr := backupCorruptFile(path); bakErr != nil {
			return fmt.Errorf("快照版本 %d 高于当前支持版本 %d（且备份失败: %v）", snap.Version, snapshotVersion, bakErr)
		}
		return fmt.Errorf("快照版本 %d 高于当前支持版本 %d（已备份到 %s.bak）", snap.Version, snapshotVersion, path)
	}

	s.ensureMaps(&snap)
	s.Restore(&snap)
	return nil
}

// ensureMaps 将缺失的 map 初始化为空 map，避免后续写入时 nil map 崩溃。
func (s *Store) ensureMaps(snap *Snapshot) {
	if snap.Elevators == nil {
		snap.Elevators = map[string]*domain.Elevator{}
	}
	if snap.Reports == nil {
		snap.Reports = map[string]*domain.StateReport{}
	}
	if snap.Events == nil {
		snap.Events = map[string]*domain.EntrapmentEvent{}
	}
	if snap.Disposals == nil {
		snap.Disposals = map[string]*domain.DisposalRecord{}
	}
	if snap.FaultLogs == nil {
		snap.FaultLogs = map[string]*domain.FaultCodeLog{}
	}
	if snap.Audits == nil {
		snap.Audits = map[string]*domain.AuditLog{}
	}
	if snap.Observations == nil {
		snap.Observations = map[string]*EntrapmentObservation{}
	}
}

// backupCorruptFile 将损坏文件备份到 <path>.bak（优先 rename，失败则复制）。
// 备份后原文件被移走，避免下一轮 Load 再次解析同一损坏文件。
func backupCorruptFile(path string) error {
	bak := path + ".bak"
	// 优先 rename：原子的把损坏文件移到 .bak，等价于“备份 + 移走原文件”。
	if err := os.Rename(path, bak); err == nil {
		return nil
	}
	// rename 失败（跨设备等）则退化为读取 + 写入 .bak，并删除原文件。
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(bak, data, 0o644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return nil
}
