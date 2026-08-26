// Package domain 定义电梯物联网诊断服务的核心领域模型与业务规则。
//
// 本文件集中定义全项目共享的枚举/常量。前端（web/）中与之对应的
// 常量定义在 web/constants.js，二者必须保持一致（README 中列有对照表）。
package domain

import (
	"fmt"
	"strings"
)

// EventStatus 困人事件状态机状态。
//
// 合法迁移关系：
//
//	alerted → accepted / escalated
//	accepted → processing / escalated
//	processing → released / escalated
//	released → （终态）
//	escalated → （终态）
type EventStatus string

const (
	// EventAlerted 已告警：困人事件刚生成，等待维保人员接单。
	EventAlerted EventStatus = "alerted"
	// EventAccepted 已接单：维保人员已接单，进入处置时限倒计时。
	EventAccepted EventStatus = "accepted"
	// EventProcessing 处置中：维保人员已在现场开展处置。
	EventProcessing EventStatus = "processing"
	// EventReleased 已解除：人员安全撤出，事件关闭。
	EventReleased EventStatus = "released"
	// EventEscalated 已升级：接单超时或现场情况恶化，事件升级并二次告警。
	EventEscalated EventStatus = "escalated"
)

// eventTransitions 记录困人事件状态机的合法迁移表。
var eventTransitions = map[EventStatus][]EventStatus{
	EventAlerted:    {EventAccepted, EventEscalated},
	EventAccepted:   {EventEscalated},
	EventProcessing: {EventReleased, EventEscalated},
	EventReleased:   {},
	EventEscalated:  {},
}

// eventStatusLabels 提供各状态的中文展示名。
var eventStatusLabels = map[EventStatus]string{
	EventAlerted:    "已告警",
	EventAccepted:   "已接单",
	EventProcessing: "处置中",
	EventReleased:   "已解除",
	EventEscalated:  "已升级",
}

// Valid 判断 EventStatus 是否为已知枚举值。
func (s EventStatus) Valid() bool {
	_, ok := eventStatusLabels[s]
	return ok
}

// Label 返回状态的中文展示名；未知状态返回原值。
func (s EventStatus) Label() string {
	if v, ok := eventStatusLabels[s]; ok {
		return v
	}
	return string(s)
}

// CanTransition 判断从当前状态是否可合法迁移到目标状态。
func (s EventStatus) CanTransition(to EventStatus) bool {
	for _, next := range eventTransitions[s] {
		if next == to {
			return true
		}
	}
	return false
}

// Transitions 返回从当前状态可合法迁移的所有目标状态。
func (s EventStatus) Transitions() []EventStatus {
	out := make([]EventStatus, len(eventTransitions[s]))
	copy(out, eventTransitions[s])
	return out
}

// IsTerminal 判断当前状态是否为终态（已解除/已升级）。
func (s EventStatus) IsTerminal() bool {
	return s == EventReleased || s == EventEscalated
}

// IsOpen 判断事件是否仍处于开放（未闭环）状态。
func (s EventStatus) IsOpen() bool {
	return !s.IsTerminal()
}

// ParseEventStatus 解析字符串为 EventStatus，失败返回错误。
func ParseEventStatus(raw string) (EventStatus, error) {
	s := EventStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !s.Valid() {
		return "", fmt.Errorf("未知的困人事件状态: %q", raw)
	}
	return s, nil
}

// DoorStatus 轿厢门状态。
type DoorStatus string

const (
	DoorOpen   DoorStatus = "open"
	DoorClosed DoorStatus = "closed"
)

var doorStatusLabels = map[DoorStatus]string{
	DoorOpen:   "开门",
	DoorClosed: "关门",
}

// Valid 判断 DoorStatus 是否合法。
func (d DoorStatus) Valid() bool {
	_, ok := doorStatusLabels[d]
	return ok
}

// Label 返回中文展示名。
func (d DoorStatus) Label() string {
	if v, ok := doorStatusLabels[d]; ok {
		return v
	}
	return string(d)
}

// ParseDoorStatus 解析字符串为 DoorStatus。
func ParseDoorStatus(raw string) (DoorStatus, error) {
	d := DoorStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !d.Valid() {
		return "", fmt.Errorf("未知的门状态: %q", raw)
	}
	return d, nil
}

// Direction 电梯运行方向。
type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
	DirectionIdle Direction = "idle"
)

var directionLabels = map[Direction]string{
	DirectionUp:   "上行",
	DirectionDown: "下行",
	DirectionIdle: "静止",
}

// Valid 判断 Direction 是否合法。
func (d Direction) Valid() bool {
	_, ok := directionLabels[d]
	return ok
}

// Label 返回中文展示名。
func (d Direction) Label() string {
	if v, ok := directionLabels[d]; ok {
		return v
	}
	return string(d)
}

// Moving 判断电梯是否处于运行状态（非静止）。
func (d Direction) Moving() bool {
	return d == DirectionUp || d == DirectionDown
}

// ParseDirection 解析字符串为 Direction。
func ParseDirection(raw string) (Direction, error) {
	d := Direction(strings.ToLower(strings.TrimSpace(raw)))
	if !d.Valid() {
		return "", fmt.Errorf("未知的运行方向: %q", raw)
	}
	return d, nil
}

// FaultType 故障类型：已知 / 未知。
type FaultType string

const (
	FaultKnown   FaultType = "known"
	FaultUnknown FaultType = "unknown"
)

// Valid 判断 FaultType 是否合法。
func (f FaultType) Valid() bool {
	return f == FaultKnown || f == FaultUnknown
}

// Label 返回中文展示名。
func (f FaultType) Label() string {
	if f == FaultKnown {
		return "已知故障"
	}
	return "未知故障"
}

// ParseFaultType 解析字符串为 FaultType。
func ParseFaultType(raw string) (FaultType, error) {
	f := FaultType(strings.ToLower(strings.TrimSpace(raw)))
	if !f.Valid() {
		return "", fmt.Errorf("未知的故障类型: %q", raw)
	}
	return f, nil
}

// PassengerSignal 乘客信号：警铃 / 红外探测 的组合。
type PassengerSignal string

const (
	// PassengerNone 无乘客信号。
	PassengerNone PassengerSignal = "none"
	// PassengerAlarm 仅警铃被按下。
	PassengerAlarm PassengerSignal = "alarm"
	// PassengerInfrared 仅红外探测到乘客。
	PassengerInfrared PassengerSignal = "infrared"
	// PassengerBoth 警铃与红外同时触发。
	PassengerBoth PassengerSignal = "both"
)

var passengerLabels = map[PassengerSignal]string{
	PassengerNone:     "无乘客信号",
	PassengerAlarm:    "警铃触发",
	PassengerInfrared: "红外探测",
	PassengerBoth:     "警铃+红外",
}

// Valid 判断 PassengerSignal 是否合法。
func (p PassengerSignal) Valid() bool {
	_, ok := passengerLabels[p]
	return ok
}

// Present 判断是否存在乘客信号（有人被困）。
func (p PassengerSignal) Present() bool {
	return p == PassengerAlarm || p == PassengerInfrared || p == PassengerBoth
}

// Label 返回中文展示名。
func (p PassengerSignal) Label() string {
	if v, ok := passengerLabels[p]; ok {
		return v
	}
	return string(p)
}

// ParsePassengerSignal 解析字符串为 PassengerSignal。
func ParsePassengerSignal(raw string) (PassengerSignal, error) {
	p := PassengerSignal(strings.ToLower(strings.TrimSpace(raw)))
	if !p.Valid() {
		return "", fmt.Errorf("未知的乘客信号: %q", raw)
	}
	return p, nil
}
