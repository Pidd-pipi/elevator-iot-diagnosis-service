package httpapi

import (
	"net/http"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/service"
)

// ReportHandler 状态上报接口。
type ReportHandler struct {
	svc *service.Services
}

// NewReportHandler 构造上报处理器。
func NewReportHandler(svc *service.Services) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// ingestRequest 状态上报请求体。
type ingestRequest struct {
	Floor          int        `json:"floor"`
	Position       string     `json:"position"`
	Direction      string     `json:"direction"`
	Door           string     `json:"door"`
	Leveling       *bool      `json:"leveling"`
	FaultCode      string     `json:"fault_code"`
	Passenger      string     `json:"passenger_signal"`
	AlarmActive    bool       `json:"alarm_active"`
	InfraredActive bool       `json:"infrared_active"`
	ReportedAt     *time.Time `json:"reported_at"`
}

// Ingest POST /api/elevators/{id}/states 状态上报。
//
// 命中业务链路：合法性校验 → 故障诊断 → 困人判定。
func (h *ReportHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req ingestRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Fail(w, r, domain.NewValidationError("body", "请求体 JSON 解析失败: "+err.Error()))
		return
	}
	if req.Floor < 0 {
		Fail(w, r, domain.NewValidationError("floor", "楼层不能为负数"))
		return
	}

	direction, err := domain.ParseDirection(req.Direction)
	if err != nil {
		Fail(w, r, domain.NewValidationError("direction", err.Error()))
		return
	}
	door, err := domain.ParseDoorStatus(req.Door)
	if err != nil {
		Fail(w, r, domain.NewValidationError("door", err.Error()))
		return
	}
	passenger, err := domain.ParsePassengerSignal(req.Passenger)
	if err != nil {
		Fail(w, r, domain.NewValidationError("passenger_signal", err.Error()))
		return
	}
	if req.FaultCode != "" {
		if err := domain.FormatFaultCode(req.FaultCode); err != nil {
			Fail(w, r, domain.NewValidationError("fault_code", err.Error()))
			return
		}
	}

	leveling := false
	if req.Leveling != nil {
		leveling = *req.Leveling
	}
	var reportedAt time.Time
	if req.ReportedAt != nil {
		reportedAt = *req.ReportedAt
	}

	report := &domain.StateReport{
		ElevatorID:     id,
		Floor:          req.Floor,
		Position:       req.Position,
		Direction:      direction,
		Door:           door,
		Leveling:       leveling,
		FaultCode:      req.FaultCode,
		Passenger:      passenger,
		AlarmActive:    req.AlarmActive,
		InfraredActive: req.InfraredActive,
		ReportedAt:     reportedAt,
	}

	result, err := h.svc.Ingest.Ingest(report)
	if err != nil {
		Fail(w, r, err)
		return
	}
	Created(w, r, result)
}
