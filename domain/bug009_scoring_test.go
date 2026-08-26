package domain

import "testing"

// 无乘客信号的关门非平层上报不得判定为困人条件。
func TestBug009EntrapmentRequiresPassenger(t *testing.T) {
	r := &StateReport{Door: DoorClosed, Leveling: false, Passenger: PassengerNone}
	if r.EntrapmentCondition() {
		t.Fatal("无乘客信号时不应满足困人条件")
	}
}
