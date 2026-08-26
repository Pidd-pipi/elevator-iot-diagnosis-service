package store

import (
	"sort"
	"sync"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// ElevatorStore 电梯台账仓储（内存 map + 读写锁）。
type ElevatorStore struct {
	mu      sync.RWMutex
	records map[string]*domain.Elevator
}

// NewElevatorStore 构造电梯仓储。
func NewElevatorStore() *ElevatorStore {
	return &ElevatorStore{records: make(map[string]*domain.Elevator)}
}

// Save 新增或覆盖电梯台账。
func (s *ElevatorStore) Save(e *domain.Elevator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = NewID("elevator")
	}
	s.records[e.ID] = e
}

// Get 按 ID 查询电梯，不存在返回 false。
// 返回深拷贝，避免调用方直接修改仓储内部对象。
func (s *ElevatorStore) Get(id string) (*domain.Elevator, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.records[id]
	if !ok {
		return nil, false
	}
	return e.Clone(), true
}

// List 返回全部电梯，按 ID 排序保证输出稳定。
func (s *ElevatorStore) List() []*domain.Elevator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Elevator, 0, len(s.records))
	for _, e := range s.records {
		out = append(out, e.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListWatchlisted 返回重点关注（低评分）电梯。
func (s *ElevatorStore) ListWatchlisted() []*domain.Elevator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Elevator, 0)
	for _, e := range s.records {
		if e.Watchlisted {
			out = append(out, e.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HealthScore < out[j].HealthScore })
	return out
}

// Count 返回电梯总数。
func (s *ElevatorStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// All 返回记录快照（用于持久化）。
func (s *ElevatorStore) All() map[string]*domain.Elevator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.Elevator, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复记录。
//
// 复制入参 map 而非直接持有其引用，避免后续写入污染调用方快照
// （恢复后继续加电梯不应改动原快照数据）。
func (s *ElevatorStore) Restore(records map[string]*domain.Elevator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]*domain.Elevator, len(records))
	for k, v := range records {
		out[k] = v
	}
	s.records = out
}
