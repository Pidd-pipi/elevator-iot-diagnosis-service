package store

import (
	"sort"
	"sync"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// DisposalStore 处置任务仓储。
type DisposalStore struct {
	mu      sync.RWMutex
	records map[string]*domain.DisposalRecord
}

// NewDisposalStore 构造处置任务仓储。
func NewDisposalStore() *DisposalStore {
	return &DisposalStore{records: make(map[string]*domain.DisposalRecord)}
}

// Save 新增或更新处置任务。
func (s *DisposalStore) Save(d *domain.DisposalRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.ID == "" {
		d.ID = NewID("disposal")
	}
	s.records[d.ID] = d
}

// Get 按 ID 查询，返回深拷贝。
func (s *DisposalStore) Get(id string) (*domain.DisposalRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.records[id]
	if !ok {
		return nil, false
	}
	return d.Clone(), true
}

// GetByEvent 按事件 ID 查询唯一处置任务，返回深拷贝。
func (s *DisposalStore) GetByEvent(eventID string) (*domain.DisposalRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.records {
		if d.EventID == eventID {
			return d.Clone(), true
		}
	}
	return nil, false
}

// List 返回全部处置任务，按接单时间倒序。
func (s *DisposalStore) List() []*domain.DisposalRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.DisposalRecord, 0, len(s.records))
	for _, d := range s.records {
		out = append(out, d.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AcceptedAt.After(out[j].AcceptedAt) })
	return out
}

// CountUntimelyByElevatorSince 统计窗口内某电梯未按时处置的次数。
// 判定口径：处置任务最终闭环（released/escalated）且 Timely=false。
func (s *DisposalStore) CountUntimelyByElevatorSince(elevatorID string, since time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, d := range s.records {
		if d.ElevatorID != elevatorID {
			continue
		}
		if !d.Status.IsTerminal() {
			continue
		}
		if d.UpdatedAt.Before(since) {
			continue
		}
		if !d.Timely {
			n++
		}
	}
	return n
}

// CountClosedByElevatorSince 统计窗口内某电梯已闭环处置次数。
func (s *DisposalStore) CountClosedByElevatorSince(elevatorID string, since time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, d := range s.records {
		if d.ElevatorID == elevatorID && d.Status.IsTerminal() && !d.UpdatedAt.Before(since) {
			n++
		}
	}
	return n
}

// All 返回记录快照。
func (s *DisposalStore) All() map[string]*domain.DisposalRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.DisposalRecord, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
// 兜底 nil map，避免历史快照缺字段后写入崩溃。
func (s *DisposalStore) Restore(records map[string]*domain.DisposalRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if records == nil {
		records = make(map[string]*domain.DisposalRecord)
	}
	s.records = records
}
