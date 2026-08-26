package domain

import "testing"

func TestComputeScore(t *testing.T) {
	// 无故障、按时处置 → 满分。
	score, watch := ComputeScore(0, 0, 2, 5, 60)
	if score != 100 || watch {
		t.Fatalf("期望 100/非重点关注，得到 %d/%v", score, watch)
	}
	// 5 次故障 + 1 次未按时：100 - 10 - 5 = 85。
	score, watch = ComputeScore(5, 1, 2, 5, 60)
	if score != 85 || watch {
		t.Fatalf("期望 85/非重点关注，得到 %d/%v", score, watch)
	}
	// 20 次故障 + 2 次未按时：100 - 40 - 10 = 50 → 重点关注。
	score, watch = ComputeScore(20, 2, 2, 5, 60)
	if score != 50 || !watch {
		t.Fatalf("期望 50/重点关注，得到 %d/%v", score, watch)
	}
	// 扣分下限为 0。
	score, _ = ComputeScore(100, 100, 2, 5, 60)
	if score != 0 {
		t.Fatalf("评分下限应为 0，得到 %d", score)
	}
}
