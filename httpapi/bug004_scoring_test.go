package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// 业务动作必须留下审计记录（升级事件应出现在审计列表）。
func TestBug004AuditLogRecordsBusinessAction(t *testing.T) {
	srv := newTestServer(t)
	// 取 seed 中已接单事件并升级。
	_, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=accepted", nil)
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("seed 应包含已接单事件")
	}
	id := events[0].(map[string]any)["id"].(string)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/escalate", map[string]any{"reason": "现场恶化"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("升级应返回 200，得到 %d", resp.StatusCode)
	}
	// 审计列表应包含 event.escalate。
	logsResp, logsPayload := doJSON(t, http.MethodGet, srv.URL+"/api/audit-logs", nil)
	if logsResp.StatusCode != http.StatusOK {
		t.Fatalf("审计接口应返回 200，得到 %d", logsResp.StatusCode)
	}
	ldata, _ := logsPayload["data"].(map[string]any)
	logs, _ := ldata["logs"].([]any)
	found := false
	for _, l := range logs {
		m := l.(map[string]any)
		if m["action"] == "event.escalate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("升级后审计列表缺少 event.escalate 记录")
	}
}

// 审计列表必须按 limit 分页，不能一次返回全部。
func TestBug004AuditListRespectsLimit(t *testing.T) {
	srv := newTestServer(t)
	// 制造多条写请求审计。
	for i := 0; i < 3; i++ {
		doJSON(t, http.MethodPost, srv.URL+"/api/events/event-not-exist/accept", nil)
	}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/audit-logs?limit=1", nil)
	if err != nil {
		t.Fatal(err)
	}
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
	logs, _ := data["logs"].([]any)
	if len(logs) > 1 {
		t.Fatalf("limit=1 时应返回不超过 1 条审计，得到 %d 条", len(logs))
	}
}
