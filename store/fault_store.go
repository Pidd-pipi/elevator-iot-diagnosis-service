package store

import (
	"sort"
	"sync"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// FaultStore 故障码记录仓储。
type FaultStore struct {
	mu      sync.RWMutex
	records map[string]*domain.FaultCodeLog
}

// NewFaultStore 构造故障码记录仓储。
func NewFaultStore() *FaultStore {
	return &FaultStore{records: make(map[string]*domain.FaultCodeLog)}
}

// Append 登记一条故障码记录。
func (s *FaultStore) Append(f *domain.FaultCodeLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.ID == "" {
		f.ID = NewID("fault")
	}
	s.records[f.ID] = f
}

// Get 按 ID 查询，返回副本。
func (s *FaultStore) Get(id string) (*domain.FaultCodeLog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.records[id]
	if !ok {
		return nil, false
	}
	return f.Clone(), true
}

// ListByElevator 按电梯查询故障记录，按发生时间倒序。
func (s *FaultStore) ListByElevator(elevatorID string, limit int) []*domain.FaultCodeLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.FaultCodeLog, 0)
	for _, f := range s.records {
		if f.ElevatorID == elevatorID {
			items = append(items, f.Clone())
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// ListUnknown 返回全部未知故障记录，按发生时间倒序。
func (s *FaultStore) ListUnknown(limit int) []*domain.FaultCodeLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.FaultCodeLog, 0)
	for _, f := range s.records {
		if !f.Known {
			items = append(items, f.Clone())
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// CountByElevatorSince 统计窗口内某电梯故障次数。
func (s *FaultStore) CountByElevatorSince(elevatorID string, since time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, f := range s.records {
		if f.ElevatorID == elevatorID && !f.OccurredAt.Before(since) {
			n++
		}
	}
	return n
}

// CountUnknown 统计未知故障总数。
func (s *FaultStore) CountUnknown() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, f := range s.records {
		if !f.Known {
			n++
		}
	}
	return n
}

// All 返回记录快照。
func (s *FaultStore) All() map[string]*domain.FaultCodeLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.FaultCodeLog, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
func (s *FaultStore) Restore(records map[string]*domain.FaultCodeLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}
