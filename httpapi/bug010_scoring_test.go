package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// 给不存在的电梯上报状态必须返回 404，不能返回成功。
func TestBug010MissingElevatorIngest404(t *testing.T) {
	srv := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"floor": 3, "position": "3F", "direction": "idle", "door": "closed",
		"leveling": true, "passenger_signal": "none",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-NOPE/states", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在的电梯上报应返回 404，得到 %d", resp.StatusCode)
	}
}

// 携带故障码的上报必须登记故障诊断记录，不能静默丢弃。
func TestBug010FaultDiagnosisRecorded(t *testing.T) {
	srv := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"floor": 1, "position": "1F", "direction": "idle", "door": "closed",
		"leveling": true, "passenger_signal": "none", "fault_code": "E01",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("正常上报应返回 201，得到 %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	diag, ok := data["diagnosis"].(map[string]any)
	if !ok {
		t.Fatal("携带故障码的上报应返回故障诊断记录，实际为空")
	}
	if diag["fault_code"] != "E01" {
		t.Fatalf("诊断记录故障码应为 E01，得到 %v", diag["fault_code"])
	}
}

// 总览的最近上报列表必须按 limit 截断，不能无限返回。
func TestBug010OverviewRecentReportsLimited(t *testing.T) {
	srv := newTestServer(t)
	// 12 条上报。
	for i := 0; i < 12; i++ {
		body, _ := json.Marshal(map[string]any{
			"floor": 3, "position": "3F", "direction": "idle", "door": "closed",
			"leveling": true, "passenger_signal": "none",
		})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/overview", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	recent, _ := data["recent_reports"].([]any)
	if len(recent) > 10 {
		t.Fatalf("最近上报列表应不超过 10 条，得到 %d 条", len(recent))
	}
}

// 上报时间戳为历史时，今日上报统计必须按接收时间（今天）计算。
func TestBug010ReportsTodayCountsStaleReports(t *testing.T) {
	srv := newTestServer(t)
	countToday := func() float64 {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/overview", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		data, _ := payload["data"].(map[string]any)
		stats, _ := data["stats"].(map[string]any)
		n, _ := stats["reports_today"].(float64)
		return n
	}
	before := countToday()
	body, _ := json.Marshal(map[string]any{
		"floor": 3, "position": "3F", "direction": "idle", "door": "closed",
		"leveling": true, "passenger_signal": "none", "reported_at": "2020-01-01T00:00:00+08:00",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	after := countToday()
	// 上报接收时间是今天，今日上报数应 +1。
	if after != before+1 {
		t.Fatalf("历史时间戳的上报应按接收时间计入今日上报：%v -> %v", before, after)
	}
}
