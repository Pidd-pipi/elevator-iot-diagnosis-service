package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func bug009Post(t *testing.T, srvURL, elevatorID string, body map[string]any) int {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, srvURL+"/api/elevators/"+elevatorID+"/states", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func bug009OpenCount(t *testing.T, srvURL string) float64 {
	t.Helper()
	resp, payload := doJSON(t, http.MethodGet, srvURL+"/api/overview", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("总览接口应返回 200，得到 %d", resp.StatusCode)
	}
	data, _ := payload["data"].(map[string]any)
	stats, _ := data["stats"].(map[string]any)
	open, _ := stats["open_events"].(float64)
	return open
}

func bug009Trapped(floor int, at string) map[string]any {
	return map[string]any{
		"floor": floor, "position": "3F-4F 之间", "direction": "idle", "door": "closed",
		"leveling": false, "passenger_signal": "alarm", "reported_at": at,
	}
}

// bug009Times 生成 5 秒间隔的 RFC3339 时间序列。
func bug009Times(n int, base string) []string {
	t, err := time.Parse(time.RFC3339, base)
	if err != nil {
		panic(err)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = t.Add(time.Duration(i*5) * time.Second).Format(time.RFC3339)
	}
	return out
}

// 困人条件中断后必须重新计时，单条上报不得立刻触发困人事件。
func TestBug009ConditionBreakResetsObservation(t *testing.T) {
	srv := newTestServer(t)
	before := bug009OpenCount(t, srv.URL)
	// 6 条困人上报（30 秒），未触发。
	times := bug009Times(6, "2026-08-26T10:00:00+08:00")
	for i := 0; i < 6; i++ {
		bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(3, times[i]))
	}
	// 条件中断（开门平层）。
	bug009Post(t, srv.URL, "ELEV-002", map[string]any{
		"floor": 3, "position": "3F", "direction": "idle", "door": "open",
		"leveling": true, "passenger_signal": "none", "reported_at": "2026-08-26T10:00:31+08:00",
	})
	// 再一条困人上报，应重新计时，不触发。
	bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(3, "2026-08-26T10:00:36+08:00"))
	if got := bug009OpenCount(t, srv.URL); got != before {
		t.Fatalf("条件中断后单条上报不应触发困人事件：%v -> %v", before, got)
	}
}

// 上报中断超过两个周期后必须重新计时。
func TestBug009GapResetsChain(t *testing.T) {
	srv := newTestServer(t)
	before := bug009OpenCount(t, srv.URL)
	times := bug009Times(6, "2026-08-26T10:00:00+08:00")
	for i := 0; i < 6; i++ {
		bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(3, times[i]))
	}
	// 中断超过两周期（40 秒后）再上报，应重新计时。
	bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(4, "2026-08-26T10:00:41+08:00"))
	if got := bug009OpenCount(t, srv.URL); got != before {
		t.Fatalf("中断超过两周期后单条上报不应触发困人事件：%v -> %v", before, got)
	}
	_ = before
}

// 事件触发并解除后，单条持续上报不得立刻重新触发。
func TestBug009NoImmediateRetriggerAfterResolve(t *testing.T) {
	srv := newTestServer(t)
	before := bug009OpenCount(t, srv.URL)
	// 8 条困人上报触发事件。
	times := bug009Times(8, "2026-08-26T10:00:00+08:00")
	for i := 0; i < 8; i++ {
		bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(3, times[i]))
	}
	if got := bug009OpenCount(t, srv.URL); got != before+1 {
		t.Fatalf("应触发 1 个困人事件，当前 %v", got)
	}
	// 取事件并解除。
	_, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=alerted", nil)
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("应有已告警事件")
	}
	id := events[0].(map[string]any)["id"].(string)
	doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/accept", nil)
	doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/resolve", map[string]any{
		"disposer": "王工", "measure": "更换门锁", "recovery_time": "2026-08-26T12:00:00+08:00",
	})
	// 单条持续困人上报，不应立刻重新触发。
	bug009Post(t, srv.URL, "ELEV-002", bug009Trapped(3, "2026-08-26T10:00:41+08:00"))
	if got := bug009OpenCount(t, srv.URL); got != before {
		t.Fatalf("解除后单条上报不应立刻重新触发：%v -> %v", before, got)
	}
}

// 无乘客信号不得判定为困人。
func TestBug009EntrapmentRequiresPassenger(t *testing.T) {
	srv := newTestServer(t)
	before := bug009OpenCount(t, srv.URL)
	times := bug009Times(8, "2026-08-26T10:00:00+08:00")
	for i := 0; i < 8; i++ {
		body := bug009Trapped(3, times[i])
		body["passenger_signal"] = "none"
		bug009Post(t, srv.URL, "ELEV-002", body)
	}
	if got := bug009OpenCount(t, srv.URL); got != before {
		t.Fatalf("无乘客信号不应触发困人事件：%v -> %v", before, got)
	}
}
