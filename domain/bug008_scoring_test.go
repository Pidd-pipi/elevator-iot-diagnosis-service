package domain

import (
	"testing"
	"time"
)

// 已接单事件必须能进入处置中（转换表不能缺边）。
func TestBug008ProcessingTransitionAllowed(t *testing.T) {
	if !EventAccepted.CanTransition(EventProcessing) {
		t.Fatal("已接单事件应可迁移到处置中，转换表缺边")
	}
}

// 升级事件必须标记二次告警已发送。
func TestBug008EscalateSendsSecondAlarm(t *testing.T) {
	now := timeNow()
	ev := NewEntrapmentEvent("e1", "ELEV-001", now, now, 40, "r1")
	if err := ev.Accept(now); err != nil {
		t.Fatal(err)
	}
	if err := ev.Escalate(now, "超时"); err != nil {
		t.Fatal(err)
	}
	if !ev.SecondAlarmSent {
		t.Fatal("升级后二次告警标志应为 true")
	}
	if ev.EscalationCount != 1 {
		t.Fatalf("升级次数应为 1，得到 %d", ev.EscalationCount)
	}
}

// 处置任务升级必须累计升级次数。
func TestBug008DisposalMarkEscalatedCounts(t *testing.T) {
	d := NewDisposalRecord("d1", "e1", "ELEV-001", timeNow())
	d.MarkEscalated(timeNow())
	if d.EscalationCount != 1 {
		t.Fatalf("处置升级次数应为 1，得到 %d", d.EscalationCount)
	}
	if d.Timely {
		t.Fatal("升级处置应标记为未按时")
	}
}

func timeNow() time.Time {
	return time.Now()
}

// 解除事件必须记录结束时间。
func TestBug008ReleaseRecordsEndedAt(t *testing.T) {
	now := timeNow()
	ev := NewEntrapmentEvent("e1", "ELEV-001", now, now, 40, "r1")
	_ = ev.Accept(now)
	if err := ev.Release(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if ev.EndedAt == nil {
		t.Fatal("解除事件应记录结束时间")
	}
}

// 升级事件必须记录结束时间。
func TestBug008EscalateRecordsEndedAt(t *testing.T) {
	now := timeNow()
	ev := NewEntrapmentEvent("e1", "ELEV-001", now, now, 40, "r1")
	_ = ev.Accept(now)
	if err := ev.Escalate(now, "超时"); err != nil {
		t.Fatal(err)
	}
	if ev.EndedAt == nil {
		t.Fatal("升级事件应记录结束时间")
	}
}

// 按时闭环的处置必须标记为按时。
func TestBug008CompleteSetsTimely(t *testing.T) {
	accepted := timeNow()
	d := NewDisposalRecord("d1", "e1", "ELEV-001", accepted)
	if err := d.Complete("王工", "更换门锁", "", accepted.Add(5*time.Minute), 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if !d.Timely {
		t.Fatal("时限内闭环的处置应标记为按时")
	}
}

// 闭环处置必须保留备注。
func TestBug008CompleteKeepsNote(t *testing.T) {
	accepted := timeNow()
	d := NewDisposalRecord("d1", "e1", "ELEV-001", accepted)
	if err := d.Complete("王工", "更换门锁", "现场排查正常", accepted.Add(5*time.Minute), 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if d.Note != "现场排查正常" {
		t.Fatalf("闭环处置备注丢失，得到 %q", d.Note)
	}
}
