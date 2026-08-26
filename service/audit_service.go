package service

import (
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// AuditService 审计日志服务：所有业务动作留痕的统一入口。
type AuditService struct {
	store  *store.Store
	logger *slog.Logger
}

// NewAuditService 构造审计服务。
func NewAuditService(st *store.Store, logger *slog.Logger) *AuditService {
	return &AuditService{store: st, logger: logger}
}

// Record 记录一条业务审计日志并落盘到审计仓储。
//
// request id 从当前请求作用域容器（CurrentTrace）读取：HTTP 请求触发的业务
// 动作会带上网关 trace id；非请求上下文（扫描任务、种子数据）下为空。
func (s *AuditService) Record(action, actor, targetType, targetID, detail string, at time.Time) *domain.AuditLog {
	log := domain.NewAuditLog(action, actor, targetType, targetID, detail, at)
	log.ID = store.NewID("audit")
	log.RequestID = CurrentTrace().RequestID()
	s.store.Audits.Append(log)
	s.logger.Info("audit", "action", action, "actor", actor, "target_type", targetType, "target_id", targetID, "detail", detail, "request_id", log.RequestID)
	return log
}

// List 返回最近 N 条审计日志。
func (s *AuditService) List(limit int) []*domain.AuditLog {
	return s.store.Audits.List(limit)
}

// ListByEvent 返回某困人事件的处置轨迹。
func (s *AuditService) ListByEvent(eventID string) []*domain.AuditLog {
	return s.store.Audits.ListByTarget("event", eventID, 50)
}
