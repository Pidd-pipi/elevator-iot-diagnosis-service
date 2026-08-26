package store

import (
	"sync"
	"time"
)

// ObservationStore 困人条件观测仓储（按电梯维度）。
type ObservationStore struct {
	mu      sync.RWMutex
	records map[string]*EntrapmentObservation
}

// NewObservationStore 构造观测仓储。
func NewObservationStore() *ObservationStore {
	return &ObservationStore{records: make(map[string]*EntrapmentObservation)}
}

// Get 查询某电梯的观测记录，返回深拷贝。
func (s *ObservationStore) Get(elevatorID string) (*EntrapmentObservation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.records[elevatorID]
	if !ok {
		return nil, false
	}
	return o.Clone(), true
}

// Set 保存观测记录。
func (s *ObservationStore) Set(o *EntrapmentObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[o.ElevatorID] = o
}

// Reset 清空某电梯的观测记录（条件中断时调用）。
func (s *ObservationStore) Reset(elevatorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, elevatorID)
}

// All 返回记录快照。
func (s *ObservationStore) All() map[string]*EntrapmentObservation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*EntrapmentObservation, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
func (s *ObservationStore) Restore(records map[string]*EntrapmentObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}

// Elapsed 返回某电梯当前累计的连续困人秒数（无观测返回 0）。
func (s *ObservationStore) Elapsed(elevatorID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.records[elevatorID]
	if !ok {
		return 0
	}
	return o.ConsecutiveSeconds
}

// IsActive 判断某电梯是否处于持续观测中。
func (s *ObservationStore) IsActive(elevatorID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.records[elevatorID]
	return ok && o.Active
}

// Since 返回某电梯本轮观测起点。
func (s *ObservationStore) Since(elevatorID string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.records[elevatorID]
	if !ok {
		return time.Time{}, false
	}
	return o.FirstSeenAt, true
}
