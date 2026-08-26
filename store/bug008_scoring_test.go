package store

import (
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// 已升级（闭环）事件不得出现在开放事件列表中。
func TestBug008EscalatedNotListedOpen(t *testing.T) {
	s := NewEventStore()
	now := time.Now()
	ev := domain.NewEntrapmentEvent("e1", "ELEV-001", now, now, 40, "r1")
	_ = ev.Accept(now)
	_ = ev.Escalate(now, "超时")
	s.Save(ev)
	open := s.ListOpen()
	for _, e := range open {
		if e.ID == "e1" {
			t.Fatal("已升级事件不应出现在开放事件列表中")
		}
	}
}

// 升级事件不得计入已解除统计（按状态精确计数）。
func TestBug008CountByStatusExact(t *testing.T) {
	s := NewEventStore()
	now := time.Now()
	ev := domain.NewEntrapmentEvent("e1", "ELEV-001", now, now, 40, "r1")
	_ = ev.Accept(now)
	_ = ev.Escalate(now, "超时")
	s.Save(ev)
	if got := s.CountByStatus(domain.EventEscalated); got != 1 {
		t.Fatalf("升级事件计数应为 1，得到 %d", got)
	}
	if got := s.CountByStatus(domain.EventReleased); got != 0 {
		t.Fatalf("升级事件不应计入已解除，得到 %d", got)
	}
}
