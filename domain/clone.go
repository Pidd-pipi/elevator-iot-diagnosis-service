package domain

import "time"

// 以下 Clone 方法用于 store 层在读操作时返回深拷贝，避免把仓储内部的
// 可变对象指针暴露给 service/handler 后，在并发读写下产生数据竞态。

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

// Clone 返回电梯台账的深拷贝。
func (e *Elevator) Clone() *Elevator {
	if e == nil {
		return nil
	}
	c := *e
	c.LastReportAt = cloneTime(e.LastReportAt)
	return &c
}

// Clone 返回困人事件的深拷贝。
func (e *EntrapmentEvent) Clone() *EntrapmentEvent {
	if e == nil {
		return nil
	}
	c := *e
	c.AcceptedAt = cloneTime(e.AcceptedAt)
	c.ProcessingAt = cloneTime(e.ProcessingAt)
	c.ReleasedAt = cloneTime(e.ReleasedAt)
	c.EscalatedAt = cloneTime(e.EscalatedAt)
	c.EndedAt = cloneTime(e.EndedAt)
	return &c
}

// Clone 返回处置任务的深拷贝。
func (d *DisposalRecord) Clone() *DisposalRecord {
	if d == nil {
		return nil
	}
	c := *d
	c.ProcessingAt = cloneTime(d.ProcessingAt)
	c.RecoveryTime = cloneTime(d.RecoveryTime)
	return &c
}

// Clone 返回状态上报的副本。
func (r *StateReport) Clone() *StateReport {
	if r == nil {
		return nil
	}
	c := *r
	return &c
}

// Clone 返回故障码记录的副本。
func (f *FaultCodeLog) Clone() *FaultCodeLog {
	if f == nil {
		return nil
	}
	c := *f
	return &c
}

// Clone 返回审计日志的副本。
func (a *AuditLog) Clone() *AuditLog {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}
