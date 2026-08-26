package store

import "sync"

// Store 聚合全部子仓储，作为服务层与持久化层之间的统一入口。
//
// persistMu 串行化 Save/Load，避免并发落盘与恢复造成快照交错；
// 各子仓储内部仍使用各自的 RWMutex 保护读写。
type Store struct {
	persistMu sync.Mutex

	Elevators    *ElevatorStore
	Reports      *ReportStore
	Events       *EventStore
	Disposals    *DisposalStore
	Faults       *FaultStore
	Audits       *AuditStore
	Observations *ObservationStore
}

// NewStore 构造一个全新的空仓储集合。
func NewStore() *Store {
	return &Store{
		Elevators:    NewElevatorStore(),
		Reports:      NewReportStore(),
		Events:       NewEventStore(),
		Disposals:    NewDisposalStore(),
		Faults:       NewFaultStore(),
		Audits:       NewAuditStore(),
		Observations: NewObservationStore(),
	}
}

// Snapshot 收集全部仓储的快照，用于 JSON 持久化。
func (s *Store) Snapshot() *Snapshot {
	return &Snapshot{
		Elevators:    s.Elevators.All(),
		Reports:      s.Reports.All(),
		Events:       s.Events.All(),
		Disposals:    s.Disposals.All(),
		FaultLogs:    s.Faults.All(),
		Audits:       s.Audits.All(),
		Observations: s.Observations.All(),
	}
}

// Restore 用快照覆盖全部仓储。
func (s *Store) Restore(snap *Snapshot) {
	s.Elevators.Restore(snap.Elevators)
	s.Reports.Restore(snap.Reports)
	s.Disposals.Restore(snap.Disposals)
	s.Faults.Restore(snap.FaultLogs)
	s.Audits.Restore(snap.Audits)
	s.Observations.Restore(snap.Observations)
}
