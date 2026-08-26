package domain

import (
	"testing"
	"time"
)

func TestDisposalValidateCompletion(t *testing.T) {
	now := time.Now()
	d := NewDisposalRecord("d-1", "e-1", "ELEV-001", now)
	if err := d.ValidateCompletion(); err == nil {
		t.Fatal("空处置信息应校验失败")
	}
	d.Disposer = "王工"
	if err := d.ValidateCompletion(); err == nil {
		t.Fatal("缺少处理措施应校验失败")
	}
	d.Measure = "更换门锁触点"
	if err := d.ValidateCompletion(); err == nil {
		t.Fatal("缺少恢复时间应校验失败")
	}
	rt := now.Add(5 * time.Minute)
	d.RecoveryTime = &rt
	if err := d.ValidateCompletion(); err != nil {
		t.Fatalf("完整处置信息应通过校验: %v", err)
	}
}

func TestDisposalTimely(t *testing.T) {
	now := time.Now()
	deadline := 10 * time.Minute
	d := NewDisposalRecord("d-1", "e-1", "ELEV-001", now)
	recovered := now.Add(8 * time.Minute)
	if err := d.Complete("王工", "处理", "", recovered, deadline); err != nil {
		t.Fatal(err)
	}
	if !d.Timely {
		t.Fatal("8 分钟内解除应判定按时")
	}
	d2 := NewDisposalRecord("d-2", "e-2", "ELEV-001", now)
	recovered2 := now.Add(11 * time.Minute)
	if err := d2.Complete("李工", "处理", "", recovered2, deadline); err != nil {
		t.Fatal(err)
	}
	if d2.Timely {
		t.Fatal("11 分钟解除应判定未按时")
	}
	if !d2.Escaped(deadline) {
		t.Fatal("Escaped 判定错误")
	}
}

func TestDisposalMarkEscalated(t *testing.T) {
	now := time.Now()
	d := NewDisposalRecord("d-1", "e-1", "ELEV-001", now)
	d.MarkEscalated(now.Add(time.Minute))
	if d.Status != EventEscalated {
		t.Fatalf("升级后状态应为 escalated，得到 %s", d.Status)
	}
	if d.EscalationCount != 1 || d.Timely {
		t.Fatal("升级处置应计为未按时且升级次数 +1")
	}
}
