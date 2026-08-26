package store

import "time"

// EntrapmentObservation 困人条件的持续观测记录（按电梯维度）。
//
// 终端每 5 秒上报一次，服务端累计「非平层 + 门关闭 + 有乘客信号」的
// 持续时长，达到阈值后生成困人事件。观测状态本身不入库，仅保存在内存，
// 但随 JSON 快照一并持久化，保证重启后可恢复观测进度。
type EntrapmentObservation struct {
	ElevatorID string `json:"elevator_id"`
	// Active 当前是否处于满足条件的持续观测中。
	Active bool `json:"active"`
	// FirstSeenAt 本轮连续观测的起点。
	FirstSeenAt time.Time `json:"first_seen_at"`
	// LastReportAt 最近一次满足条件的上报时间。
	LastReportAt time.Time `json:"last_report_at"`
	// ConsecutiveSeconds 已累计的持续秒数。
	ConsecutiveSeconds int `json:"consecutive_seconds"`
	// ReportIDs 本轮观测涉及的上报 ID。
	ReportIDs []string `json:"report_ids,omitempty"`
}
