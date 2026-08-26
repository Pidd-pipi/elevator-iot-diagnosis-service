package domain

import (
	"fmt"
	"time"
)

// EntrapmentEvent 困人事件实体。
//
// 判定规则：同一电梯在「非平层 + 门关闭 + 存在乘客信号」状态持续超过
// 阈值（默认 30 秒）后生成困人事件；同一电梯存在未关闭事件时不重复生成。
// 事件状态机见 enums.go 中的 EventStatus 迁移表。
type EntrapmentEvent struct {
	ID string `json:"id"`
	// ElevatorID 发生困人的电梯。
	ElevatorID string `json:"elevator_id"`
	// Status 事件当前状态。
	Status EventStatus `json:"status"`
	// FirstSeenAt 首次观测到满足困人条件的时间。
	FirstSeenAt time.Time `json:"first_seen_at"`
	// StartedAt 事件正式生成（告警）时间。
	StartedAt time.Time `json:"started_at"`
	// AlertedAt 告警发出时间，与 StartedAt 相同。
	AlertedAt time.Time `json:"alerted_at"`
	// AcceptedAt 接单时间。
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	// ProcessingAt 开始处置时间。
	ProcessingAt *time.Time `json:"processing_at,omitempty"`
	// ReleasedAt 解除（关闭）时间。
	ReleasedAt *time.Time `json:"released_at,omitempty"`
	// EscalatedAt 升级时间。
	EscalatedAt *time.Time `json:"escalated_at,omitempty"`
	// EndedAt 事件结束时间（解除或升级）。
	EndedAt *time.Time `json:"ended_at,omitempty"`
	// EscalationCount 升级次数。
	EscalationCount int `json:"escalation_count"`
	// SecondAlarmSent 是否已发送二次告警。
	SecondAlarmSent bool `json:"second_alarm_sent"`
	// DurationSec 满足困人条件累计时长（秒），用于展示。
	DurationSec int `json:"duration_sec"`
	// Description 事件描述。
	Description string `json:"description"`
	// LatestReportID 最近一次触发/维持该事件的原始上报 ID。
	LatestReportID string `json:"latest_report_id,omitempty"`
	// CreatedAt 记录创建时间。
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 记录最近更新时间。
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEntrapmentEvent 生成新的困人事件（初始状态为已告警）。
func NewEntrapmentEvent(id, elevatorID string, firstSeenAt, now time.Time, durationSec int, reportID string) *EntrapmentEvent {
	return &EntrapmentEvent{
		ID:             id,
		ElevatorID:     elevatorID,
		Status:         EventAlerted,
		FirstSeenAt:    firstSeenAt,
		StartedAt:      now,
		AlertedAt:      now,
		DurationSec:    durationSec,
		Description:    fmt.Sprintf("电梯 %s 疑似困人：非平层停梯且门关闭，乘客信号持续 %d 秒", elevatorID, durationSec),
		LatestReportID: reportID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CanTransition 判断是否可迁移到目标状态。
func (e *EntrapmentEvent) CanTransition(to EventStatus) bool {
	return e.Status.CanTransition(to)
}

// IsOpen 判断事件是否未闭环。
func (e *EntrapmentEvent) IsOpen() bool {
	return e.Status.IsOpen()
}

// Accept 接单：alerted → accepted。
func (e *EntrapmentEvent) Accept(at time.Time) error {
	if e.Status != EventAlerted {
		return fmt.Errorf("%w: 仅已告警事件可接单，当前状态 %s", ErrInvalidState, e.Status)
	}
	e.Status = EventAccepted
	t := at
	e.AcceptedAt = &t
	e.UpdatedAt = at
	return nil
}

// StartProcessing 开始处置：accepted → processing。
func (e *EntrapmentEvent) StartProcessing(at time.Time) error {
	if e.Status != EventAccepted {
		return fmt.Errorf("%w: 仅已接单事件可开始处置，当前状态 %s", ErrInvalidState, e.Status)
	}
	e.Status = EventProcessing
	t := at
	e.ProcessingAt = &t
	e.UpdatedAt = at
	return nil
}

// Release 解除：processing → released（已接单未处置时也允许直接解除）。
func (e *EntrapmentEvent) Release(at time.Time) error {
	if e.Status != EventProcessing && e.Status != EventAccepted {
		return fmt.Errorf("%w: 仅处置中/已接单事件可解除，当前状态 %s", ErrInvalidState, e.Status)
	}
	if e.Status == EventAccepted {
		t := at
		e.ProcessingAt = &t
	}
	e.Status = EventReleased
	t := at
	e.ReleasedAt = &t
	e.UpdatedAt = at
	return nil
}

// Escalate 升级：alerted/accepted/processing → escalated，并发送二次告警。
func (e *EntrapmentEvent) Escalate(at time.Time, reason string) error {
	if !e.Status.IsOpen() {
		return fmt.Errorf("%w: 事件已闭环，无法升级（当前状态 %s）", ErrInvalidState, e.Status)
	}
	e.Status = EventEscalated
	t := at
	e.EscalatedAt = &t
	e.EscalationCount++
	e.Description = fmt.Sprintf("%s；已升级并发送二次告警（原因：%s）", e.Description, reason)
	e.UpdatedAt = at
	return nil
}

// RemainingDeadline 返回接单后剩余处置时限；未接单或已闭环返回 0。
func (e *EntrapmentEvent) RemainingDeadline(deadline time.Duration, now time.Time) time.Duration {
	if e.AcceptedAt == nil || e.Status.IsTerminal() {
		return 0
	}
	expire := e.AcceptedAt.Add(deadline)
	if now.After(expire) {
		return 0
	}
	return expire.Sub(now)
}

// IsOverdue 判断事件是否已超过接单处置时限。
func (e *EntrapmentEvent) IsOverdue(deadline time.Duration, now time.Time) bool {
	if e.AcceptedAt == nil || !e.Status.IsOpen() {
		return false
	}
	return now.After(e.AcceptedAt.Add(deadline))
}
