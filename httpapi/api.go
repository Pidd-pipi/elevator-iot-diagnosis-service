package httpapi

import (
	"log/slog"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/service"
)

// API 聚合 httpapi 层所需的全部依赖。
type API struct {
	svc    *service.Services
	cfg    *config.Config
	logger *slog.Logger
}

// NewAPI 构造 API 聚合对象。
func NewAPI(svc *service.Services, cfg *config.Config, logger *slog.Logger) *API {
	return &API{svc: svc, cfg: cfg, logger: logger}
}
