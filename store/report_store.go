package store

import (
	"sort"
	"sync"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// ReportStore 状态上报仓储。
type ReportStore struct {
	mu      sync.RWMutex
	records map[string]*domain.StateReport
}

// NewReportStore 构造上报仓储。
func NewReportStore() *ReportStore {
	return &ReportStore{records: make(map[string]*domain.StateReport)}
}

// Append 保存一条状态上报。
func (s *ReportStore) Append(r *domain.StateReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = NewID("report")
	}
	s.records[r.ID] = r
}

// Get 按 ID 查询，返回副本。
func (s *ReportStore) Get(id string) (*domain.StateReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	if !ok {
		return nil, false
	}
	return r.Clone(), true
}

// ListByElevator 按电梯查询上报，按上报时间倒序，支持 limit。
func (s *ReportStore) ListByElevator(elevatorID string, limit int) []*domain.StateReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.StateReport, 0)
	for _, r := range s.records {
		if r.ElevatorID == elevatorID {
			items = append(items, r.Clone())
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ReportedAt.After(items[j].ReportedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// LatestByElevator 返回某电梯最近一条上报。
func (s *ReportStore) LatestByElevator(elevatorID string) (*domain.StateReport, bool) {
	list := s.ListByElevator(elevatorID, 1)
	if len(list) == 0 {
		return nil, false
	}
	return list[0], true
}

// CountBetween 统计窗口内某电梯的上报条数。
func (s *ReportStore) CountBetween(elevatorID string, from, to time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, r := range s.records {
		if r.ElevatorID == elevatorID && !r.ReportedAt.Before(from) && !r.ReportedAt.After(to) {
			n++
		}
	}
	return n
}

// CountToday 统计今日全量上报条数（用于总览）。
func (s *ReportStore) CountToday(now time.Time) int {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, r := range s.records {
		if !r.CreatedAt.Before(dayStart) {
			n++
		}
	}
	return n
}

// All 返回记录快照。
func (s *ReportStore) All() map[string]*domain.StateReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.StateReport, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
func (s *ReportStore) Restore(records map[string]*domain.StateReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}
