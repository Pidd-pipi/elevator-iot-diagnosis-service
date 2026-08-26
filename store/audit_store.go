package store

import (
	"sort"
	"sync"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// AuditStore 审计日志仓储。
type AuditStore struct {
	mu      sync.RWMutex
	records map[string]*domain.AuditLog
}

// NewAuditStore 构造审计日志仓储。
func NewAuditStore() *AuditStore {
	return &AuditStore{records: make(map[string]*domain.AuditLog)}
}

// Append 追加一条审计日志。
func (s *AuditStore) Append(a *domain.AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = NewID("audit")
	}
	s.records[a.ID] = a
}

// List 返回全部审计日志，按时间倒序，支持 limit。
func (s *AuditStore) List(limit int) []*domain.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.AuditLog, 0, len(s.records))
	for _, a := range s.records {
		items = append(items, a.Clone())
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// ListByTarget 按目标类型与目标 ID 过滤。
func (s *AuditStore) ListByTarget(targetType, targetID string, limit int) []*domain.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*domain.AuditLog, 0)
	for _, a := range s.records {
		if a.TargetType == targetType && a.TargetID == targetID {
			items = append(items, a.Clone())
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// Count 返回审计日志总数。
func (s *AuditStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// All 返回记录快照。
func (s *AuditStore) All() map[string]*domain.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*domain.AuditLog, len(s.records))
	for k, v := range s.records {
		out[k] = v
	}
	return out
}

// Restore 从快照恢复。
func (s *AuditStore) Restore(records map[string]*domain.AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}
