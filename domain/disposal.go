package domain

import (
	"time"
)

// DisposalRecord 处置任务实体，与困人事件一一对应（事件 ID 唯一关联）。
//
// 关闭（解除）处置任务前必须完整填写：处置人、处理措施、恢复时间，
// 否则不允许关闭 —— 规则 6。
type DisposalRecord struct {
	ID string `json:"id"`
	// EventID 关联的困人事件 ID。
	EventID string `json:"event_id"`
	// ElevatorID 关联电梯。
	ElevatorID string `json:"elevator_id"`
	// Status 镜像事件状态，便于按处置维度统计。
	Status EventStatus `json:"status"`
	// AcceptedAt 接单时间。
	AcceptedAt time.Time `json:"accepted_at"`
	// ProcessingAt 开始处置时间。
	ProcessingAt *time.Time `json:"processing_at,omitempty"`
	// Disposer 处置人（关闭任务时必填）。
	Disposer string `json:"disposer"`
	// Measure 处理措施（关闭任务时必填）。
	Measure string `json:"measure"`
	// RecoveryTime 恢复时间（关闭任务时必填）。
	RecoveryTime *time.Time `json:"recovery_time,omitempty"`
	// Note 备注。
	Note string `json:"note"`
	// EscalationCount 升级次数。
	EscalationCount int `json:"escalation_count"`
	// Timely 是否按时处置（接单到闭环 ≤ AcceptDeadline）。
	Timely bool `json:"timely"`
	// CreatedAt 记录创建时间。
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 记录最近更新时间。
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDisposalRecord 构造与事件关联的处置任务。
func NewDisposalRecord(id, eventID, elevatorID string, acceptedAt time.Time) *DisposalRecord {
	return &DisposalRecord{
		ID:         id,
		EventID:    eventID,
		ElevatorID: elevatorID,
		Status:     EventAccepted,
		AcceptedAt: acceptedAt,
		CreatedAt:  acceptedAt,
		UpdatedAt:  acceptedAt,
	}
}

// ValidateCompletion 校验关闭处置任务所需字段是否完整。
// 处置人、处理措施、恢复时间任一缺失即不允许关闭。
func (d *DisposalRecord) ValidateCompletion() error {
	if d.Disposer == "" {
		return NewValidationError("disposer", "处置人必填")
	}
	if d.Measure == "" {
		return NewValidationError("measure", "处理措施必填")
	}
	if d.RecoveryTime == nil {
		return NewValidationError("recovery_time", "恢复时间必填")
	}
	return nil
}

// Complete 关闭处置任务：写入处置信息并标记是否按时。
func (d *DisposalRecord) Complete(disposer, measure, note string, recoveryTime time.Time, deadline time.Duration) error {
	d.Disposer = disposer
	d.Measure = measure
	d.Note = note
	t := recoveryTime
	d.RecoveryTime = &t
	d.Status = EventReleased
	d.Timely = !recoveryTime.After(d.AcceptedAt.Add(deadline))
	d.UpdatedAt = recoveryTime
	return nil
}

// MarkEscalated 标记处置任务升级（未按时，二次告警）。
func (d *DisposalRecord) MarkEscalated(at time.Time) {
	d.Status = EventEscalated
	d.EscalationCount++
	d.Timely = false
	d.UpdatedAt = at
}

// StartProcessing 记录开始处置时间。
func (d *DisposalRecord) StartProcessing(at time.Time) {
	if d.ProcessingAt == nil {
		t := at
		d.ProcessingAt = &t
	}
	d.Status = EventProcessing
	d.UpdatedAt = at
}

// Escaped 判断该处置是否超出接单时限。
func (d *DisposalRecord) Escaped(deadline time.Duration) bool {
	if d.RecoveryTime == nil {
		return false
	}
	return d.RecoveryTime.After(d.AcceptedAt.Add(deadline))
}
