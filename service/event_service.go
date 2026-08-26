package service

import (
	"fmt"
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// EventService 困人事件处置服务：接单 → 处置 → 解除 / 升级。
type EventService struct {
	store  *store.Store
	cfg    *config.Config
	logger *slog.Logger
	audit  *AuditService
}

// NewEventService 构造处置服务。
func NewEventService(st *store.Store, cfg *config.Config, logger *slog.Logger) *EventService {
	return &EventService{store: st, cfg: cfg, logger: logger}
}

// SetAudit 注入审计服务（由 Services 装配时调用）。
func (s *EventService) SetAudit(a *AuditService) {
	s.audit = a
}

// List 按过滤条件查询困人事件。
func (s *EventService) List(filter store.EventFilter) []*domain.EntrapmentEvent {
	return s.store.Events.List(filter)
}

// Get 查询单个困人事件。
func (s *EventService) Get(id string) (*domain.EntrapmentEvent, error) {
	e, ok := s.store.Events.Get(id)
	if !ok {
		return nil, fmt.Errorf("%w: 困人事件 %s", domain.ErrNotFound, id)
	}
	return e, nil
}

// GetDisposal 查询事件关联的处置任务。
func (s *EventService) GetDisposal(eventID string) (*domain.DisposalRecord, error) {
	d, ok := s.store.Disposals.GetByEvent(eventID)
	if !ok {
		return nil, fmt.Errorf("%w: 事件 %s 暂无处置任务", domain.ErrNotFound, eventID)
	}
	return d, nil
}

// Accept 接单处置：alerted → accepted。
//
// 规则 3：接单后进入 10 分钟处置时限倒计时，超时由扫描任务自动升级。
func (s *EventService) Accept(eventID, actor string, at time.Time) (*domain.EntrapmentEvent, error) {
	event, err := s.Get(eventID)
	if err != nil {
		return nil, err
	}
	if err := event.Accept(at); err != nil {
		return nil, err
	}

	disposal, ok := s.store.Disposals.GetByEvent(eventID)
	if !ok {
		disposal = domain.NewDisposalRecord(store.NewID("disposal"), eventID, event.ElevatorID, at)
	}
	disposal.Status = domain.EventAccepted
	disposal.UpdatedAt = at
	s.store.Disposals.Save(disposal)
	s.store.Events.Save(event)

	s.audit.Record("event.accept", actor, "event", eventID,
		fmt.Sprintf("电梯 %s 困人事件接单，处置时限 %s", event.ElevatorID, s.cfg.AcceptDeadline), at)
	return event, nil
}

// StartProcessing 开始处置：accepted → processing。
func (s *EventService) StartProcessing(eventID, actor string, at time.Time) (*domain.EntrapmentEvent, error) {
	event, err := s.Get(eventID)
	if err != nil {
		return nil, err
	}
	if err := event.StartProcessing(at); err != nil {
		return nil, err
	}
	if d, ok := s.store.Disposals.GetByEvent(eventID); ok {
		d.StartProcessing(at)
		s.store.Disposals.Save(d)
	}
	s.store.Events.Save(event)
	s.audit.Record("event.processing", actor, "event", eventID,
		fmt.Sprintf("电梯 %s 困人事件开始现场处置", event.ElevatorID), at)
	return event, nil
}

// ResolveRequest 解除处置请求参数。
type ResolveRequest struct {
	Disposer     string `json:"disposer"`
	Measure      string `json:"measure"`
	Note         string `json:"note"`
	RecoveryTime string `json:"recovery_time"` // RFC3339
}

// Resolve 处置完成：accepted/processing → released。
//
// 规则 6：处置人、处理措施、恢复时间必须完整填写，否则不允许关闭处置任务。
func (s *EventService) Resolve(eventID, actor string, req ResolveRequest, at time.Time) (*domain.EntrapmentEvent, error) {
	event, err := s.Get(eventID)
	if err != nil {
		return nil, err
	}
	if !event.IsOpen() {
		return nil, fmt.Errorf("%w: 事件已闭环，无法处置完成（当前状态 %s）", domain.ErrInvalidState, event.Status)
	}

	recoveryTime, err := parseRecoveryTime(req.RecoveryTime)
	if err != nil {
		return nil, err
	}

	// 校验处置任务必填字段（规则 6）：处置人、处理措施、恢复时间。
	if req.Disposer == "" {
		return nil, domain.NewValidationError("disposer", "处置人必填")
	}
	if req.Measure == "" {
		return nil, domain.NewValidationError("measure", "处理措施必填")
	}
	if req.RecoveryTime == "" {
		return nil, domain.NewValidationError("recovery_time", "恢复时间必填")
	}

	disposal, ok := s.store.Disposals.GetByEvent(eventID)
	if !ok {
		disposal = domain.NewDisposalRecord(store.NewID("disposal"), eventID, event.ElevatorID, at)
	}
	// 用领域对象的完整校验兜底（复用 DisposalRecord.ValidateCompletion）。
	check := &domain.DisposalRecord{
		Disposer:     req.Disposer,
		Measure:      req.Measure,
		RecoveryTime: &recoveryTime,
	}
	if err := check.ValidateCompletion(); err != nil {
		return nil, err
	}

	// 状态机迁移：accepted → processing → released。
	if err := event.Release(at); err != nil {
		return nil, err
	}
	if err := disposal.Complete(req.Disposer, req.Measure, req.Note, recoveryTime, s.cfg.AcceptDeadline); err != nil {
		return nil, err
	}
	s.store.Events.Save(event)
	s.store.Disposals.Save(disposal)

	// 处置闭环后刷新电梯健康评分。
	detail, err := ComputeScoreFor(s.store, s.cfg, event.ElevatorID, at)
	if err != nil {
		s.logger.Warn("refresh score failed", "elevator", event.ElevatorID, "err", err)
	} else {
		s.logger.Info("score refreshed after resolve", "elevator", event.ElevatorID, "score", detail.Score)
	}

	s.audit.Record("event.resolve", actor, "event", eventID,
		fmt.Sprintf("电梯 %s 困人解除：处置人 %s，措施 %s，按时=%v",
			event.ElevatorID, req.Disposer, req.Measure, disposal.Timely), at)
	return event, nil
}

// Escalate 升级处置：alerted/accepted/processing → escalated（二次告警）。
func (s *EventService) Escalate(eventID, actor, reason string, at time.Time) (*domain.EntrapmentEvent, error) {
	event, err := s.Get(eventID)
	if err != nil {
		return nil, err
	}
	if !event.IsOpen() {
		return nil, fmt.Errorf("%w: 事件已闭环，无法升级（当前状态 %s）", domain.ErrInvalidState, event.Status)
	}
	if err := event.Escalate(at, reason); err != nil {
		return nil, err
	}
	if d, ok := s.store.Disposals.GetByEvent(eventID); ok {
		d.MarkEscalated(at)
		s.store.Disposals.Save(d)
	}
	s.store.Events.Save(event)

	detail, err := ComputeScoreFor(s.store, s.cfg, event.ElevatorID, at)
	if err != nil {
		s.logger.Warn("refresh score failed", "elevator", event.ElevatorID, "err", err)
	} else {
		s.logger.Info("score refreshed after escalate", "elevator", event.ElevatorID, "score", detail.Score)
	}

	s.audit.Record("event.escalate", actor, "event", eventID,
		fmt.Sprintf("电梯 %s 困人事件升级（第 %d 次），发送二次告警，原因：%s",
			event.ElevatorID, event.EscalationCount, reason), at)
	return event, nil
}

// AutoEscalate 超时自动升级（供扫描任务调用）。
func (s *EventService) AutoEscalate(eventID, reason string, at time.Time) (*domain.EntrapmentEvent, error) {
	return s.Escalate(eventID, "system", reason, at)
}

// parseRecoveryTime 解析恢复时间（RFC3339）。
func parseRecoveryTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: 恢复时间格式非法（应为 RFC3339）", domain.ErrValidation)
	}
	return t, nil
}
