// Package service 承载全部业务用例：采集、困人判定、处置流转、
// 故障诊断、健康评分、超时扫描与审计。
package service

import (
	"log/slog"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/store"
)

// Services 聚合全部业务服务，供 httpapi 与 main 统一装配。
type Services struct {
	Store    *store.Store
	Config   *config.Config
	Logger   *slog.Logger
	Ingest   *IngestService
	Events   *EventService
	Diagnose *DiagnoseService
	Scoring  *ScoringService
	Sweeper  *OverdueSweeper
	Audit    *AuditService
	Seed     *SeedService
}

// NewServices 构造服务集合，并建立各服务之间的依赖。
func NewServices(st *store.Store, cfg *config.Config, logger *slog.Logger) *Services {
	audit := NewAuditService(st, logger)
	ingest := NewIngestService(st, cfg, logger)
	events := NewEventService(st, cfg, logger)
	events.SetAudit(audit)
	diagnose := NewDiagnoseService(st, logger)
	scoring := NewScoringService(st, cfg, logger)
	sweeper := NewOverdueSweeper(st, cfg, logger, events)
	seed := NewSeedService(st, cfg, ingest)
	return &Services{
		Store:    st,
		Config:   cfg,
		Logger:   logger,
		Ingest:   ingest,
		Events:   events,
		Diagnose: diagnose,
		Scoring:  scoring,
		Sweeper:  sweeper,
		Audit:    audit,
		Seed:     seed,
	}
}
