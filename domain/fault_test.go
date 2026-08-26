package domain

import (
	"testing"
	"time"
)

func TestLookupFaultRule(t *testing.T) {
	rule, ok := LookupFaultRule("E01")
	if !ok {
		t.Fatal("E01 应为已知故障码")
	}
	if rule.Name != "门锁回路故障" {
		t.Fatalf("E01 名称错误: %s", rule.Name)
	}
	if !rule.Known {
		t.Fatal("已知故障码 Known 应为 true")
	}
	if _, ok := LookupFaultRule("X99"); ok {
		t.Fatal("X99 应为未知故障码")
	}
}

func TestUnknownFaultDiagnosis(t *testing.T) {
	at := time.Now()
	log := UnknownFaultDiagnosis("ELEV-001", "X77", "report-1", at, "未知故障码，需人工确认")
	if log.Known {
		t.Fatal("未知故障记录 Known 应为 false")
	}
	if log.FaultType != FaultUnknown {
		t.Fatalf("未知故障类型应为 unknown，得到 %s", log.FaultType)
	}
	if log.Diagnosis == "" {
		t.Fatal("未知故障必须给出人工确认提示")
	}
}

func TestFormatFaultCode(t *testing.T) {
	if err := FormatFaultCode("E01"); err != nil {
		t.Fatalf("合法故障码校验失败: %v", err)
	}
	if err := FormatFaultCode(""); err != nil {
		t.Fatalf("空故障码应允许: %v", err)
	}
	// 未知故障码允许登记（不得被白名单拒绝）。
	if err := FormatFaultCode("X77"); err != nil {
		t.Fatalf("未知故障码不应被格式校验拒绝: %v", err)
	}
	for _, bad := range []string{"E01 E02", "AB-1", "123456789"} {
		if err := FormatFaultCode(bad); err == nil {
			t.Errorf("非法故障码 %q 应校验失败", bad)
		}
	}
}

func TestDefaultFaultRulesCount(t *testing.T) {
	rules := DefaultFaultRules()
	if len(rules) < 8 {
		t.Fatalf("内置故障码映射表过少: %d", len(rules))
	}
	for _, r := range rules {
		if r.Code == "" || r.Diagnosis == "" || r.Suggestion == "" {
			t.Errorf("故障码 %q 字段不完整", r.Code)
		}
	}
}
