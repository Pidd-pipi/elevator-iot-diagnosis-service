package httpapi

import (
	"net/http"
	"testing"
)

// 查询不存在的困人事件应返回 404（错误链断裂会被误判成 500）。
func TestBug005MissingEventReturns404(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/api/events/event-not-exist", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在的事件应返回 404，得到 %d", resp.StatusCode)
	}
}

// 对已接单事件重复接单应返回 409（非法状态迁移）。
func TestBug005DoubleAcceptReturns409(t *testing.T) {
	srv := newTestServer(t)
	// seed 已生成 ELEV-001 的已接单事件，先取真实事件 ID 再重复接单。
	listResp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=accepted", nil)
	_ = listResp
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("seed 应包含已接单事件")
	}
	first := events[0].(map[string]any)
	id := first["id"].(string)
	resp2, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/accept", nil)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("重复接单应返回 409，得到 %d", resp2.StatusCode)
	}
}

// 恢复时间格式非法应返回 422（校验错误不能被吞成 500）。
func TestBug005BadRecoveryTime422(t *testing.T) {
	srv := newTestServer(t)
	_, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=accepted", nil)
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("seed 应包含已接单事件")
	}
	id := events[0].(map[string]any)["id"].(string)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/resolve", map[string]any{
		"disposer": "王工", "measure": "更换门锁", "recovery_time": "not-a-time",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("恢复时间非法应返回 422，得到 %d", resp.StatusCode)
	}
}

// 升级不存在的困人事件应返回 404。
func TestBug005EscalateMissing404(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/event-not-exist/escalate", map[string]any{"reason": "现场恶化"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("升级不存在的事件应返回 404，得到 %d", resp.StatusCode)
	}
}

// 解除已闭环事件应返回 409（非法状态迁移）。
func TestBug005ResolveClosedEvent409(t *testing.T) {
	srv := newTestServer(t)
	_, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=released", nil)
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("seed 应包含已解除事件")
	}
	id := events[0].(map[string]any)["id"].(string)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/resolve", map[string]any{
		"disposer": "王工", "measure": "更换门锁", "recovery_time": "2026-08-26T12:00:00+08:00",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("解除已闭环事件应返回 409，得到 %d", resp.StatusCode)
	}
}

// 升级已闭环事件应返回 409（非法状态迁移）。
func TestBug005EscalateClosedEvent409(t *testing.T) {
	srv := newTestServer(t)
	_, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?status=released", nil)
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("seed 应包含已解除事件")
	}
	id := events[0].(map[string]any)["id"].(string)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+id+"/escalate", map[string]any{"reason": "现场恶化"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("升级已闭环事件应返回 409，得到 %d", resp.StatusCode)
	}
}

// 接单不存在的困人事件应返回 404。
func TestBug005AcceptMissing404(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/api/events/event-not-exist/accept", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("接单不存在的事件应返回 404，得到 %d", resp.StatusCode)
	}
}
