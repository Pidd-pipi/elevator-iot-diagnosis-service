package domain

import "time"

// Elevator 电梯台账实体，贯穿 存储 → 领域 → 服务 → 接口 → 前端 全链路。
type Elevator struct {
	// ID 电梯唯一编号，如 ELEV-001。
	ID string `json:"id"`
	// Building 所属楼栋。
	Building string `json:"building"`
	// Model 电梯型号。
	Model string `json:"model"`
	// InstallDate 安装日期（yyyy-MM-dd）。
	InstallDate string `json:"install_date"`
	// CapacityKg 额定载重（千克）。
	CapacityKg int `json:"capacity_kg"`
	// Floors 服务楼层数。
	Floors int `json:"floors"`
	// HealthScore 健康评分（0-100），由评分服务定期刷新。
	HealthScore int `json:"health_score"`
	// Watchlisted 是否进入重点关注名单（评分 ≤ WatchlistThreshold）。
	Watchlisted bool `json:"watchlisted"`
	// Status 当前运行状态摘要（最近一次上报的状态机结果）。
	Status string `json:"status"`
	// LastReportAt 最近一次有效状态上报时间。
	LastReportAt *time.Time `json:"last_report_at,omitempty"`
	// CreatedAt 台账创建时间。
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 台账最近更新时间。
	UpdatedAt time.Time `json:"updated_at"`
}

// NewElevator 构造电梯台账，初始化默认评分 100。
func NewElevator(id, building, model, installDate string, capacityKg, floors int) *Elevator {
	now := time.Now()
	return &Elevator{
		ID:          id,
		Building:    building,
		Model:       model,
		InstallDate: installDate,
		CapacityKg:  capacityKg,
		Floors:      floors,
		HealthScore: 100,
		Watchlisted: false,
		Status:      "unknown",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ApplyScore 应用最新健康评分并同步重点关注标记。
func (e *Elevator) ApplyScore(score int, watchlistThreshold int) {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	e.HealthScore = score
	e.Watchlisted = score <= watchlistThreshold
	e.UpdatedAt = time.Now()
}

// TouchStatus 更新电梯的最近状态摘要。
func (e *Elevator) TouchStatus(status string, at time.Time) {
	e.Status = status
	t := at
	e.LastReportAt = &t
	e.UpdatedAt = at
}

// StatusLabel 返回当前状态摘要的中文描述。
func (e *Elevator) StatusLabel() string {
	switch e.Status {
	case "normal":
		return "运行正常"
	case "fault":
		return "故障告警"
	case "trapped":
		return "困人告警"
	case "unknown":
		return "暂无数据"
	default:
		return e.Status
	}
}
