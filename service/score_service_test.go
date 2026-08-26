package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/store"
)

func TestScoringWatchlist(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewStore()
	elevator := domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18)
	st.Elevators.Save(elevator)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)

	now := time.Now()
	// 制造 10 条近 30 天故障 → 扣 20 分 → 80 分（非重点关注）。
	for i := 0; i < 10; i++ {
		st.Faults.Append(domain.KnownFaultLog("ELEV-001",
			domain.FaultCodeRule{Code: "E01", Diagnosis: "d", Suggestion: "s", Known: true},
			"report-x", now.Add(-time.Duration(i)*time.Hour)))
	}
	detail, err := svc.Scoring.GetScore("ELEV-001")
	if err != nil {
		t.Fatalf("评分失败: %v", err)
	}
	if detail.Score != 80 {
		t.Fatalf("期望 80 分，得到 %d", detail.Score)
	}
	if detail.Watchlisted {
		t.Fatal("80 分不应进入重点关注")
	}

	// 再增加 20 条故障 → 扣 40 → 60 分 → 进入重点关注。
	for i := 0; i < 20; i++ {
		st.Faults.Append(domain.KnownFaultLog("ELEV-001",
			domain.FaultCodeRule{Code: "E03", Diagnosis: "d", Suggestion: "s", Known: true},
			"report-y", now.Add(-time.Duration(i)*time.Minute)))
	}
	detail, err = svc.Scoring.GetScore("ELEV-001")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Score != 40 {
		t.Fatalf("期望 40 分，得到 %d", detail.Score)
	}
	if !detail.Watchlisted {
		t.Fatal("40 分应进入重点关注")
	}
	elevator, _ = st.Elevators.Get("ELEV-001")
	if !elevator.Watchlisted {
		t.Fatal("电梯台账应同步重点关注标记")
	}
}

func TestScoringUntimelyDisposal(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewStore()
	st.Elevators.Save(domain.NewElevator("ELEV-001", "A 栋", "T1", "2020-01-01", 1000, 18))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewServices(st, cfg, logger)

	now := time.Now()
	// 一次未按时处置 → 扣 5 分 → 95 分。
	d := domain.NewDisposalRecord("d-1", "e-1", "ELEV-001", now.Add(-30*time.Minute))
	recovered := now.Add(-19 * time.Minute) // 11 分钟后解除 → 未按时
	if err := d.Complete("王工", "处理", "", recovered, cfg.AcceptDeadline); err != nil {
		t.Fatal(err)
	}
	d.UpdatedAt = recovered
	st.Disposals.Save(d)

	detail, err := svc.Scoring.GetScore("ELEV-001")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Score != 95 || detail.UntimelyCount != 1 {
		t.Fatalf("期望 95 分/1 次未按时，得到 %d/%d", detail.Score, detail.UntimelyCount)
	}
}
