package service

import (
	"fmt"
	"log/slog"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

// IngestResult 状态上报处理的完整结果。
type IngestResult struct {
	// Report 已入库的有效上报。
	Report *domain.StateReport `json:"report"`
	// Diagnosis 若上报携带故障码，返回诊断记录。
	Diagnosis *domain.FaultCodeLog `json:"diagnosis,omitempty"`
	// EntrapmentEvent 若本次上报触发困人事件，返回新事件。
	EntrapmentEvent *domain.EntrapmentEvent `json:"entrapment_event,omitempty"`
	// EntrapmentState 困人判定状态：none / observing / triggered。
	EntrapmentState string `json:"entrapment_state"`
	// ConsecutiveSeconds 当前累计的困人条件持续秒数。
	ConsecutiveSeconds int `json:"consecutive_seconds"`
}

// IngestService 状态采集服务。
//
// 链路：合法性校验 → 落库 → 故障诊断 → 困人判定。
type IngestService struct {
	store  *store.Store
	cfg    *config.Config
	logger *slog.Logger
}

// NewIngestService 构造采集服务。
func NewIngestService(st *store.Store, cfg *config.Config, logger *slog.Logger) *IngestService {
	return &IngestService{store: st, cfg: cfg, logger: logger}
}

// Ingest 处理一条终端状态上报。
//
// 步骤：
//  1. 校验电梯存在；
//  2. 状态机合法性校验（运行中禁止开门等），非法直接拒绝；
//  3. 落库上报，刷新电梯状态摘要；
//  4. 故障码诊断登记；
//  5. 困人条件持续观测，达到阈值且无未关闭事件时生成困人事件。
func (s *IngestService) Ingest(report *domain.StateReport) (*IngestResult, error) {
	elevator, ok := s.store.Elevators.Get(report.ElevatorID)
	if !ok {
		return nil, fmt.Errorf("%w: 电梯 %s 不存在", domain.ErrNotFound, report.ElevatorID)
	}

	// 2. 状态机合法性校验。
	if err := report.ValidateState(elevator.Floors); err != nil {
		return nil, err
	}

	// 3. 落库。
	if report.ID == "" {
		report.ID = store.NewID("report")
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now()
	}
	if report.ReportedAt.IsZero() {
		report.ReportedAt = report.CreatedAt
	}
	s.store.Reports.Append(report)
	elevator.TouchStatus(report.HealthStatus(), report.ReportedAt)
	s.store.Elevators.Save(elevator)

	result := &IngestResult{Report: report, EntrapmentState: "none"}

	// 4. 故障诊断。
	if report.FaultCode != "" {
		result.Diagnosis = s.diagnoseFor(elevator.ID, report)
	}

	// 5. 困人判定。
	s.trackEntrapment(report, result)
	return result, nil
}

// diagnoseFor 为上报中的故障码执行诊断登记。
func (s *IngestService) diagnoseFor(elevatorID string, report *domain.StateReport) *domain.FaultCodeLog {
	return s.diagnose(elevatorID, report.FaultCode, report.ID, report.ReportedAt, s.cfg.UnknownFaultPrompt)
}

// diagnose 封装诊断服务调用，便于测试注入。
func (s *IngestService) diagnose(elevatorID, faultCode, reportID string, at time.Time, prompt string) *domain.FaultCodeLog {
	if faultCode == "" {
		return nil
	}
	var log *domain.FaultCodeLog
	if rule, ok := domain.LookupFaultRule(faultCode); ok {
		log = domain.KnownFaultLog(elevatorID, rule, reportID, at)
	} else {
		log = domain.UnknownFaultDiagnosis(elevatorID, faultCode, reportID, at, prompt)
		s.logger.Warn("unknown fault code registered", "elevator", elevatorID, "code", faultCode)
	}
	log.ID = store.NewID("fault")
	s.store.Faults.Append(log)
	return log
}

// trackEntrapment 更新困人条件观测并决定是否生成困人事件。
//
// 判定规则（规则 2）：非平层 + 门关闭 + 存在乘客信号，持续超过阈值
// （默认 30 秒）即生成困人事件；同一电梯存在未关闭事件时不重复生成。
func (s *IngestService) trackEntrapment(report *domain.StateReport, result *IngestResult) {
	period := s.cfg.ReportPeriod
	threshold := s.cfg.EntrapmentThreshold

	if !report.EntrapmentCondition() {
		// 条件中断：清空观测，重新计时。
		result.EntrapmentState = "none"
		return
	}

	now := report.ReportedAt
	obs, ok := s.store.Observations.Get(report.ElevatorID)
	if !ok || !obs.Active {
		obs = &store.EntrapmentObservation{
			ElevatorID:         report.ElevatorID,
			Active:             true,
			FirstSeenAt:        now,
			LastReportAt:       now,
			ConsecutiveSeconds: 0,
		}
	} else {
		gap := now.Sub(obs.LastReportAt)
		if gap <= 0 {
			gap = period
		}
		obs.ConsecutiveSeconds += int(gap.Seconds())
		obs.LastReportAt = now
	}
	obs.ReportIDs = append(obs.ReportIDs, report.ID)
	s.store.Observations.Set(obs)

	result.ConsecutiveSeconds = obs.ConsecutiveSeconds
	if obs.ConsecutiveSeconds <= int(threshold.Seconds()) {
		result.EntrapmentState = "observing"
		return
	}

	// 达到阈值：同一电梯存在未关闭事件时不重复生成（规则 2）。
	if _, open := s.store.Events.OpenByElevator(report.ElevatorID); open {
		result.EntrapmentState = "observing"
		return
	}

	event := domain.NewEntrapmentEvent(
		store.NewID("event"),
		report.ElevatorID,
		obs.FirstSeenAt,
		now,
		obs.ConsecutiveSeconds,
		report.ID,
	)
	s.store.Events.Save(event)
	// 告警动作留痕。
	at := now
	s.auditRecord("event.alert", "system", "event", event.ID,
		fmt.Sprintf("电梯 %s 触发困人告警，持续 %d 秒", report.ElevatorID, obs.ConsecutiveSeconds), at)
	// 生成事件后重置本轮观测计时，避免重复累加。
	s.store.Observations.Set(obs)

	result.EntrapmentEvent = event
	result.EntrapmentState = "triggered"
	s.logger.Warn("entrapment event created", "event", event.ID, "elevator", report.ElevatorID)
}

// auditRecord 记录审计日志（复用 audit 服务能力，避免循环依赖）。
func (s *IngestService) auditRecord(action, actor, targetType, targetID, detail string, at time.Time) {
	log := domain.NewAuditLog(action, actor, targetType, targetID, detail, at)
	log.ID = store.NewID("audit")
	s.store.Audits.Append(log)
}

// ListRecentReports 返回最近 N 条上报（供调试/页面展示）。
func (s *IngestService) ListRecentReports(limit int) []*domain.StateReport {
	all := s.store.Reports.All()
	items := make([]*domain.StateReport, 0, len(all))
	for _, r := range all {
		items = append(items, r)
	}
	// 简单排序：按接收时间倒序。
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].CreatedAt.After(items[i].CreatedAt) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
