package domain

import (
	"fmt"
	"time"
)

// FaultCodeRule 故障码诊断映射规则。
type FaultCodeRule struct {
	// Code 故障码，如 E01。
	Code string `json:"code"`
	// Name 故障名称。
	Name string `json:"name"`
	// Category 故障类别（电气/机械/信号/安全）。
	Category string `json:"category"`
	// Severity 严重程度（low/medium/high）。
	Severity string `json:"severity"`
	// Diagnosis 诊断结论。
	Diagnosis string `json:"diagnosis"`
	// Suggestion 处理建议。
	Suggestion string `json:"suggestion"`
	// Known 是否为已知故障。
	Known bool `json:"known"`
}

// FaultCodeLog 故障码记录实体：每出现一次故障码登记一条。
type FaultCodeLog struct {
	ID         string `json:"id"`
	ElevatorID string `json:"elevator_id"`
	FaultCode  string `json:"fault_code"`
	Diagnosis  string `json:"diagnosis"`
	Suggestion string `json:"suggestion"`
	// Known 是否命中已知故障码表。
	Known bool `json:"known"`
	// FaultType known / unknown。
	FaultType FaultType `json:"fault_type"`
	// ReportID 来源状态上报。
	ReportID string `json:"report_id,omitempty"`
	// OccurredAt 故障发生时间。
	OccurredAt time.Time `json:"occurred_at"`
}

// defaultFaultRules 内置故障码诊断映射表。
var defaultFaultRules = []FaultCodeRule{
	{Code: "E01", Name: "门锁回路故障", Category: "电气", Severity: "high", Diagnosis: "门锁回路检测异常，电梯保护性停梯", Suggestion: "检查门锁触点与回路接线，必要时更换门锁", Known: true},
	{Code: "E02", Name: "平层感应器故障", Category: "信号", Severity: "medium", Diagnosis: "平层感应信号丢失，可能导致停层不准", Suggestion: "检查平层感应器安装位置与接线", Known: true},
	{Code: "E03", Name: "变频器过载", Category: "电气", Severity: "high", Diagnosis: "变频器输出过载，电机电流超限", Suggestion: "检查负载与变频器参数，排查机械卡阻", Known: true},
	{Code: "E04", Name: "抱闸制动异常", Category: "机械", Severity: "high", Diagnosis: "抱闸间隙异常或制动器未完全打开", Suggestion: "检查抱闸间隙、制动器线圈与磨损", Known: true},
	{Code: "E05", Name: "限速器动作", Category: "安全", Severity: "high", Diagnosis: "限速器触发，安全钳可能已动作", Suggestion: "由持证维保人员检查安全回路与安全钳", Known: true},
	{Code: "E06", Name: "通讯中断", Category: "信号", Severity: "medium", Diagnosis: "轿厢与控制柜通讯超时", Suggestion: "检查随行电缆与通讯板", Known: true},
	{Code: "E07", Name: "超载保护动作", Category: "安全", Severity: "low", Diagnosis: "轿厢超载，超载开关动作", Suggestion: "确认载荷并引导乘客分批乘坐", Known: true},
	{Code: "E08", Name: "门区感应故障", Category: "信号", Severity: "medium", Diagnosis: "门区信号异常，影响平层开门", Suggestion: "检查门区感应器与门区开关", Known: true},
	{Code: "E09", Name: "曳引机温度过高", Category: "机械", Severity: "medium", Diagnosis: "曳引机温升异常，可能存在润滑不足", Suggestion: "检查曳引机润滑与散热", Known: true},
	{Code: "E10", Name: "安全回路断开", Category: "安全", Severity: "high", Diagnosis: "安全回路断开，电梯紧急停梯", Suggestion: "逐段排查安全回路触点", Known: true},
	{Code: "E11", Name: "按钮卡死", Category: "电气", Severity: "low", Diagnosis: "轿内按钮信号持续有效", Suggestion: "检查按钮微动开关", Known: true},
	{Code: "E12", Name: "钢丝绳张力异常", Category: "机械", Severity: "high", Diagnosis: "钢丝绳张力偏差超限", Suggestion: "测量并调整钢丝绳张力", Known: true},
}

// DefaultFaultRules 返回内置故障码映射表的副本。
func DefaultFaultRules() []FaultCodeRule {
	out := make([]FaultCodeRule, len(defaultFaultRules))
	copy(out, defaultFaultRules)
	return out
}

// LookupFaultRule 按故障码查询诊断规则；未知故障码返回 ok=false。
func LookupFaultRule(code string) (FaultCodeRule, bool) {
	for _, r := range defaultFaultRules {
		if r.Code == code {
			return r, true
		}
	}
	return FaultCodeRule{}, false
}

// UnknownFaultDiagnosis 构造未知故障码的登记记录（不静默丢弃）。
func UnknownFaultDiagnosis(elevatorID, faultCode, reportID string, occurredAt time.Time, prompt string) *FaultCodeLog {
	return &FaultCodeLog{
		ID:         "",
		ElevatorID: elevatorID,
		FaultCode:  faultCode,
		Diagnosis:  prompt,
		Suggestion: "请安排维保人员现场确认故障码含义并补充知识库",
		Known:      false,
		FaultType:  FaultUnknown,
		ReportID:   reportID,
		OccurredAt: occurredAt,
	}
}

// KnownFaultLog 构造已知故障的登记记录。
func KnownFaultLog(elevatorID string, rule FaultCodeRule, reportID string, occurredAt time.Time) *FaultCodeLog {
	return &FaultCodeLog{
		ID:         "",
		ElevatorID: elevatorID,
		FaultCode:  rule.Code,
		Diagnosis:  rule.Diagnosis,
		Suggestion: rule.Suggestion,
		Known:      true,
		FaultType:  FaultKnown,
		ReportID:   reportID,
		OccurredAt: occurredAt,
	}
}

// SeverityLabel 返回严重程度中文标签。
func SeverityLabel(severity string) string {
	switch severity {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return severity
	}
}

// FormatFaultCode 校验故障码格式（输入卫生检查）。
//
// 注意：未知故障码（未命中知识库的编码）同样必须登记并提示人工确认，
// 因此这里只做基础格式约束（1-8 位字母/数字），不做白名单校验。
func FormatFaultCode(code string) error {
	if code == "" {
		return nil
	}
	if len(code) > 8 {
		return fmt.Errorf("故障码格式非法: %q（长度不得超过 8 位）", code)
	}
	for _, ch := range code {
		if !isAlphaNumeric(ch) {
			return fmt.Errorf("故障码格式非法: %q（仅允许字母与数字）", code)
		}
	}
	return nil
}

// isAlphaNumeric 判断字符是否为 ASCII 字母或数字。
func isAlphaNumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}
