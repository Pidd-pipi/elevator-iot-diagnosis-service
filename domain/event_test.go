package domain

import (
	"testing"
	"time"
)

func newTestEvent() *EntrapmentEvent {
	return NewEntrapmentEvent("event-1", "ELEV-001", time.Now().Add(-40*time.Second), time.Now(), 40, "report-1")
}

func TestEventStatusTransitions(t *testing.T) {
	now := time.Now()
	event := newTestEvent()
	if event.Status != EventAlerted {
		t.Fatalf("初始状态应为 alerted，得到 %s", event.Status)
	}
	if err := event.Accept(now); err != nil {
		t.Fatalf("alerted->accepted 迁移失败: %v", err)
	}
	if err := event.StartProcessing(now.Add(time.Minute)); err != nil {
		t.Fatalf("accepted->processing 迁移失败: %v", err)
	}
	if err := event.Release(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("processing->released 迁移失败: %v", err)
	}
	if event.Status != EventReleased {
		t.Fatalf("最终状态应为 released，得到 %s", event.Status)
	}
	if event.IsOpen() {
		t.Fatal("released 事件不应处于开放状态")
	}
}

func TestEventIllegalTransitions(t *testing.T) {
	now := time.Now()
	event := newTestEvent()
	// alerted 不允许直接 processing / released。
	if err := event.StartProcessing(now); err == nil {
		t.Fatal("alerted->processing 应被拒绝")
	}
	if err := event.Release(now); err == nil {
		t.Fatal("alerted->released 应被拒绝")
	}
	// 已升级事件不可再操作。
	if err := event.Escalate(now, "测试"); err != nil {
		t.Fatalf("升级失败: %v", err)
	}
	if err := event.Accept(now); err == nil {
		t.Fatal("escalated->accepted 应被拒绝")
	}
	if err := event.Escalate(now, "再次升级"); err == nil {
		t.Fatal("终态事件不可重复升级")
	}
}

func TestEventEscalateSendsSecondAlarm(t *testing.T) {
	now := time.Now()
	event := newTestEvent()
	if err := event.Accept(now); err != nil {
		t.Fatal(err)
	}
	if event.SecondAlarmSent {
		t.Fatal("升级前不应发送二次告警")
	}
	if err := event.Escalate(now.Add(11*time.Minute), "接单超时"); err != nil {
		t.Fatal(err)
	}
	if event.EscalationCount != 1 {
		t.Fatalf("升级次数应为 1，得到 %d", event.EscalationCount)
	}
	if !event.SecondAlarmSent {
		t.Fatal("升级后应发送二次告警")
	}
	if event.Status != EventEscalated {
		t.Fatalf("状态应为 escalated，得到 %s", event.Status)
	}
}

func TestEventOverdue(t *testing.T) {
	now := time.Now()
	deadline := 10 * time.Minute
	event := newTestEvent()
	if err := event.Accept(now.Add(-11 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !event.IsOverdue(deadline, now) {
		t.Fatal("接单 11 分钟应判定为超时")
	}
	event2 := newTestEvent()
	if err := event2.Accept(now.Add(-5 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if event2.IsOverdue(deadline, now) {
		t.Fatal("接单 5 分钟不应判定为超时")
	}
}

func TestParseEventStatus(t *testing.T) {
	if _, err := ParseEventStatus("alerted"); err != nil {
		t.Fatalf("合法状态解析失败: %v", err)
	}
	if _, err := ParseEventStatus("bogus"); err == nil {
		t.Fatal("非法状态应解析失败")
	}
}

func TestEventStatusCanTransition(t *testing.T) {
	cases := []struct {
		from, to EventStatus
		want     bool
	}{
		{EventAlerted, EventAccepted, true},
		{EventAlerted, EventEscalated, true},
		{EventAlerted, EventProcessing, false},
		{EventAccepted, EventProcessing, true},
		{EventAccepted, EventEscalated, true},
		{EventProcessing, EventReleased, true},
		{EventProcessing, EventEscalated, true},
		{EventReleased, EventEscalated, false},
		{EventEscalated, EventReleased, false},
	}
	for _, c := range cases {
		if got := c.from.CanTransition(c.to); got != c.want {
			t.Errorf("%s->%s 期望 %v，得到 %v", c.from, c.to, c.want, got)
		}
	}
}
