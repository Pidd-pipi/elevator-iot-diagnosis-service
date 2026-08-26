package store

import (
	"sync"
	"testing"

	"example.com/elevator-iot-diagnosis-service/domain"
)

// 并发写 + 并发读（Get）不得产生数据竞争。
func TestBug001ConcurrentSaveGetRace(t *testing.T) {
	es := NewElevatorStore()
	es.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			es.Save(domain.NewElevator("ELEV-00"+string(rune('2'+i%5)), "A 栋", "T", "2020-01-01", 1000, 18))
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			if e, ok := es.Get("ELEV-001"); ok {
				_ = e.HealthScore
			}
		}
	}()
	close(start)
	wg.Wait()
}

// 并发 Count 与 Save 不得产生数据竞争。
func TestBug001ConcurrentCountSaveRace(t *testing.T) {
	es := NewElevatorStore()
	es.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_ = es.Count()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			es.Save(domain.NewElevator("ELEV-00"+string(rune('2'+i%5)), "A 栋", "T", "2020-01-01", 1000, 18))
		}
	}()
	close(start)
	wg.Wait()
}

// Restore 后新增记录不得污染调用方持有的快照 map（深拷贝隔离）。
func TestBug001RestoreDoesNotAliasSnapshot(t *testing.T) {
	es := NewElevatorStore()
	snap := map[string]*domain.Elevator{
		"ELEV-001": domain.NewElevator("ELEV-001", "A 栋", "T", "2020-01-01", 1000, 18),
	}
	es.Restore(snap)
	es.Save(domain.NewElevator("ELEV-002", "B 栋", "T", "2020-01-01", 1000, 18))
	if _, ok := snap["ELEV-002"]; ok {
		t.Fatal("Restore 后 Save 污染了调用方快照 map（内部引用了同一 map）")
	}
}
