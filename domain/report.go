package domain

import (
	"fmt"
	"time"
)

// StateReport 终端状态上报记录实体。
//
// 终端按 5 秒周期上报：电梯编号 + 轿厢位置 + 运行方向 + 门状态 +
// 平层信号 + 故障码 + 乘客信号。服务端对每条上报做状态机合法性校验
// （如「运行中禁止开门」），非法上报直接拒绝且不落库。
type StateReport struct {
	ID         string `json:"id"`
	ElevatorID string `json:"elevator_id"`
	// Floor 轿厢当前所在楼层；非平层时可能为 0（井道中）或越界值。
	Floor int `json:"floor"`
	// Position 位置描述，如 "3F-4F 之间"。
	Position  string     `json:"position"`
	Direction Direction  `json:"direction"`
	Door      DoorStatus `json:"door"`
	// Leveling 平层信号：true 表示轿厢已平层停靠。
	Leveling bool `json:"leveling"`
	// FaultCode 故障码，空串表示无故障。
	FaultCode string `json:"fault_code"`
	// Passenger 乘客信号（警铃/红外）汇总。
	Passenger PassengerSignal `json:"passenger_signal"`
	// AlarmActive 警铃是否被按下。
	AlarmActive bool `json:"alarm_active"`
	// InfraredActive 红外是否探测到乘客。
	InfraredActive bool `json:"infrared_active"`
	// ReportedAt 终端采集时间（可透传，默认服务端接收时间）。
	ReportedAt time.Time `json:"reported_at"`
	// CreatedAt 服务端接收时间。
	CreatedAt time.Time `json:"created_at"`
}

// ValidateState 对状态上报做状态机合法性校验：
//
//  1. 方向必须合法（up/down/idle）；
//  2. 门状态必须合法（open/closed）；
//  3. 乘客信号必须合法；
//  4. 运行中（方向非 idle）禁止开门 —— 核心安全规则；
//  5. 声称平层时楼层必须落在 1..floors 范围内。
func (r *StateReport) ValidateState(floors int) error {
	if !r.Direction.Valid() {
		return NewValidationError("direction", fmt.Sprintf("非法运行方向: %q", r.Direction))
	}
	if !r.Door.Valid() {
		return NewValidationError("door", fmt.Sprintf("非法门状态: %q", r.Door))
	}
	if !r.Passenger.Valid() {
		return NewValidationError("passenger_signal", fmt.Sprintf("非法乘客信号: %q", r.Passenger))
	}
	if r.Direction.Moving() && r.Door == DoorOpen {
		return NewValidationError("door", "运行中禁止开门：电梯运行时门必须处于关闭状态")
	}
	if r.Leveling && (r.Floor < 1 || r.Floor > floors) {
		return NewValidationError("floor", fmt.Sprintf("平层状态下楼层越界: %d (有效范围 1-%d)", r.Floor, floors))
	}
	return nil
}

// HealthStatus 根据上报内容推导电梯当前健康状态摘要。
func (r *StateReport) HealthStatus() string {
	if r.FaultCode != "" {
		return "fault"
	}
	if r.Passenger.Present() && !r.Leveling && r.Door == DoorClosed {
		return "trapped"
	}
	return "normal"
}

// EntrapmentCondition 判断当前上报是否满足困人条件：
// 非平层 + 门关闭 + 存在乘客信号。
func (r *StateReport) EntrapmentCondition() bool {
	return !r.Leveling && r.Door == DoorClosed
}
