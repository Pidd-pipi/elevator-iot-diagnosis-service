package domain

import "time"

// AuditLog 操作审计日志实体。
//
// 覆盖两类来源：
//  1. 业务动作（event.accept / event.resolve / event.escalate / event.alert /
//     event.auto_escalate），由各 service 通过 audit_service 记录；
//  2. HTTP 写请求（http.request），由 middleware/audit.go 中间件记录。
type AuditLog struct {
	ID string `json:"id"`
	// Action 动作标识，如 event.accept。
	Action string `json:"action"`
	// Actor 操作者（用户/终端/系统）。
	Actor string `json:"actor"`
	// TargetType 目标类型（elevator/event/disposal/report）。
	TargetType string `json:"target_type"`
	// TargetID 目标 ID。
	TargetID string `json:"target_id"`
	// Detail 操作详情。
	Detail string `json:"detail"`
	// RequestID 关联的 HTTP trace id。
	RequestID string `json:"request_id,omitempty"`
	// CreatedAt 记录时间。
	CreatedAt time.Time `json:"created_at"`
}

// NewAuditLog 构造审计日志。
func NewAuditLog(action, actor, targetType, targetID, detail string, at time.Time) *AuditLog {
	return &AuditLog{
		Action:     action,
		Actor:      actor,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  at,
	}
}
