package store

import (
	"sort"
	"sync"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// EventFilter 困人事件查询过滤条件。
type EventFilter struct {
	// ElevatorID 按电梯过滤（空表示全部）。
	ElevatorID string
	// Status 按状态过滤（空表示全部）。
	Status domain.EventStatus
	// Limit 返回条数上限（≤0 表示不限）。
	Limit int
}

// EventStore 困人事件仓储。
type EventStore struct {
	mu      sync.RWMutex
	records map[string]*domain.EntrapmentEvent
}

// NewEventStore 构造困人事件仓储。
func NewEventStore() *EventStore {
	return &EventStore{records: make(map[string]*domain.EntrapmentEvent)}
}

// Save 新增或更新困人事件。
func (s *EventStore) Save(e *domain.EntrapmentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = NewID("event")
	}
	s.records[e.ID] = e
}

// Get 按 ID 查询，返回深拷贝。
func (s *EventStore) Get(id string) (*domain.EntrapmentEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.records[id]
	if !ok {
		return nil, false
	}
	return e.Clone(), true
}

// List 按过滤条件查询事件，按生成时间倒序。
func (s *EventStore) List(filter EventFilter) []*domain.EntrapmentEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.EntrapmentEvent, 0)
	for _, e := range s.records {
		if filter.ElevatorID != "" && e.ElevatorID != filter.ElevatorID {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		items = append(items, e.Clone())
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.After(items[j].StartedAt) })
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items
}

// ListOpen 返回全部未闭环事件（已解除/已升级均属终态，不在此列）。
func (s *EventStore) ListOpen() []*domain.EntrapmentEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.EntrapmentEvent, 0)
	for _, e := range s.records {
		if !e.IsOpen() {
			continue
		}
		items = append(items, e.Clone())
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.After(items[j].StartedAt) })
	return items
}

// OpenByElevator 返回某电梯未闭环的事件（困人判定去重用），深拷贝。
func (s *EventStore) OpenByElevator(elevatorID string) (*domain.EntrapmentEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.records {
		if e.ElevatorID == elevatorID && e.IsOpen() {
			return e.Clone(), true
		}
	}
	return nil, false
}

// CountOpen 统计未闭环事件数。
func (s *EventStore) CountOpen() int {
	return len(s.ListOpen())
}

// CountByStatus 统计指定状态的事件数。
func (s *EventStore) CountByStatus(status domain.EventStatus) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, e := range s.records {
		if e.Status == status {
			n++
		}
	}
	return n
}

// All 返回记录快照。
func (s *EventStore) All() map[string]*domain.EntrapmentEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.EntrapmentEvent, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
func (s *EventStore) Restore(records map[string]*domain.EntrapmentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}
