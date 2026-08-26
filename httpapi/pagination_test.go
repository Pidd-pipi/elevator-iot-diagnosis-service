package httpapi

import (
	"net/http"
	"testing"
)

func TestPaginationDefaultsAndTotal(t *testing.T) {
	srv := newTestServer(t)

	resp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/elevators", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("电梯列表失败: %d", resp.StatusCode)
	}
	data := payload["data"].(map[string]any)
	if data["total"].(float64) < 6 {
		t.Fatalf("total 应 ≥6，得到 %v", data["total"])
	}
	if data["limit"].(float64) != float64(defaultPageLimit) {
		t.Fatalf("默认 limit 应为 %d，得到 %v", defaultPageLimit, data["limit"])
	}

	resp, payload = doJSON(t, http.MethodGet, srv.URL+"/api/elevators?limit=2&offset=1", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("分页请求失败: %d", resp.StatusCode)
	}
	data = payload["data"].(map[string]any)
	items := data["elevators"].([]any)
	if len(items) != 2 {
		t.Fatalf("limit=2 应返回 2 条，得到 %d", len(items))
	}
	if data["offset"].(float64) != 1 {
		t.Fatalf("offset 应为 1，得到 %v", data["offset"])
	}
}

func TestPaginationRejectsInvalid(t *testing.T) {
	srv := newTestServer(t)
	for _, qs := range []string{"limit=-1", "limit=abc", "offset=-2", "offset=xyz"} {
		resp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events?"+qs, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s 应返回 422，得到 %d", qs, resp.StatusCode)
		}
		if payload["code"].(float64) != 42200 {
			t.Fatalf("%s 业务码应为 42200，得到 %v", qs, payload["code"])
		}
	}
}

func TestPaginationCapsLimit(t *testing.T) {
	srv := newTestServer(t)
	resp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/elevators?limit=99999", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("超大 limit 应被截断到上限，得到 %d", resp.StatusCode)
	}
	data := payload["data"].(map[string]any)
	if data["limit"].(float64) != float64(maxPageLimit) {
		t.Fatalf("limit 应被截断到 %d，得到 %v", maxPageLimit, data["limit"])
	}
}
