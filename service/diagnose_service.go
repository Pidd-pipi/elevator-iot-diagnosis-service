package service

import (
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// DiagnoseService 故障码诊断服务。
//
// 规则 4：故障码映射到诊断结论；未知故障码必须登记并提示人工确认，
// 不得静默丢弃。
type DiagnoseService struct {
	store  *store.Store
	logger *slog.Logger
}

// NewDiagnoseService 构造诊断服务。
func NewDiagnoseService(st *store.Store, logger *slog.Logger) *DiagnoseService {
	return &DiagnoseService{store: st, logger: logger}
}

// Rules 返回完整故障码诊断映射表。
func (s *DiagnoseService) Rules() []domain.FaultCodeRule {
	return domain.DefaultFaultRules()
}

// Diagnose 对上报中的故障码执行诊断并登记故障记录。
// 返回诊断记录；无故障码时返回 nil。
func (s *DiagnoseService) Diagnose(elevatorID, faultCode, reportID string, occurredAt time.Time, unknownPrompt string) *domain.FaultCodeLog {
	if faultCode == "" {
		return nil
	}
	var log *domain.FaultCodeLog
	if rule, ok := domain.LookupFaultRule(faultCode); ok {
		log = domain.KnownFaultLog(elevatorID, rule, reportID, occurredAt)
	} else {
		// 未知故障码：登记并提示人工确认，绝不静默丢弃。
		log = domain.UnknownFaultDiagnosis(elevatorID, faultCode, reportID, occurredAt, unknownPrompt)
		s.logger.Warn("unknown fault code registered", "elevator", elevatorID, "code", faultCode)
	}
	log.ID = store.NewID("fault")
	s.store.Faults.Append(log)
	return log
}

// ListByElevator 查询某电梯的故障码时间线。
func (s *DiagnoseService) ListByElevator(elevatorID string, limit int) []*domain.FaultCodeLog {
	return s.store.Faults.ListByElevator(elevatorID, limit)
}

// ListUnknown 查询未知故障码记录。
func (s *DiagnoseService) ListUnknown(limit int) []*domain.FaultCodeLog {
	return s.store.Faults.ListUnknown(limit)
}

// UnknownCount 统计未知故障记录数。
func (s *DiagnoseService) UnknownCount() int {
	return s.store.Faults.CountUnknown()
}
