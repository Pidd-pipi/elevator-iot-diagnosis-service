package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// 诊断规则接口返回的故障码顺序必须保持知识库原始顺序（E01 在前）。
func TestBug007RulesOrderStable(t *testing.T) {
	srv := newTestServer(t)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/diagnosis", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("诊断接口应返回 200，得到 %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	rules, _ := data["rules"].([]any)
	if len(rules) == 0 {
		t.Fatal("规则列表为空")
	}
	first := rules[0].(map[string]any)
	if first["code"] != "E01" {
		t.Fatalf("规则列表应按知识库顺序返回，首个应为 E01，得到 %v", first["code"])
	}
}

// 上报已知故障后，未知故障统计不得增加、未知列表不得混入已知故障。
func TestBug007UnknownCountExcludesKnown(t *testing.T) {
	srv := newTestServer(t)
	countUnknown := func() float64 {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/diagnosis", nil)
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
		cnt, _ := data["unknown_cnt"].(float64)
		return cnt
	}
	before := countUnknown()
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
	_ = resp.Body.Close()
	after := countUnknown()
	if after != before {
		t.Fatalf("上报已知故障后 unknown_cnt 不应变化：%v -> %v", before, after)
	}
	// 已知故障 E01 不得出现在未知列表。
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/diagnosis", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	unknown, _ := data["unknown"].([]any)
	for _, u := range unknown {
		if u.(map[string]any)["fault_code"] == "E01" {
			t.Fatal("已知故障 E01 不应出现在未知故障列表中")
		}
	}
}

// 查询不存在电梯的故障时间线应返回 404。
func TestBug007FaultsMissingElevator404(t *testing.T) {
	srv := newTestServer(t)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/elevators/ELEV-NOPE/faults", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在电梯的故障时间线应返回 404，得到 %d", resp.StatusCode)
	}
}
