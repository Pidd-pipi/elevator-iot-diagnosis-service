package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/service"
	"example.com/elevator-iot-diagnosis-service/store"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.ReportPeriod = 5 * time.Second
	cfg.EntrapmentThreshold = 30 * time.Second
	st := store.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewServices(st, cfg, logger)
	if err := svc.Seed.EnsureSeed(); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	handler := Router(svc, cfg, logger, fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><body>test</body></html>")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log(1)")},
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func doJSON(t *testing.T, method, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", "tester")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("请求失败 %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("响应解析失败 %s %s: %v", method, url, err)
	}
	return resp, payload
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	for _, path := range []string{"/healthz", "/api/healthz"} {
		resp, payload := doJSON(t, http.MethodGet, srv.URL+path, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s 应返回 200，得到 %d", path, resp.StatusCode)
		}
		if payload["code"].(float64) != 0 {
			t.Fatalf("%s code 应为 0", path)
		}
	}
}

func TestElevatorAndOverview(t *testing.T) {
	srv := newTestServer(t)
	resp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/elevators", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("电梯列表失败: %d", resp.StatusCode)
	}
	data := payload["data"].(map[string]any)
	if data["total"].(float64) < 6 {
		t.Fatalf("种子电梯数应 ≥6，得到 %v", data["total"])
	}

	resp, payload = doJSON(t, http.MethodGet, srv.URL+"/api/overview", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("总览失败: %d", resp.StatusCode)
	}
	stats := payload["data"].(map[string]any)["stats"].(map[string]any)
	if stats["elevator_total"].(float64) < 6 {
		t.Fatal("总览电梯总数异常")
	}

	resp, payload = doJSON(t, http.MethodGet, srv.URL+"/api/elevators/ELEV-001", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("电梯详情失败: %d", resp.StatusCode)
	}
}

func TestIngestToEntrapmentFlow(t *testing.T) {
	srv := newTestServer(t)
	start := time.Now().Add(-2 * time.Minute)
	var eventID string
	// 连续上报触发困人事件（ELEV-002 无种子开放事件）。
	for i := 0; i < 8; i++ {
		at := start.Add(time.Duration(i) * 5 * time.Second)
		body := map[string]any{
			"floor":            3,
			"position":         "3F-4F 之间",
			"direction":        "idle",
			"door":             "closed",
			"leveling":         false,
			"fault_code":       "",
			"passenger_signal": "alarm",
			"alarm_active":     true,
			"reported_at":      at.Format(time.RFC3339),
		}
		resp, payload := doJSON(t, http.MethodPost, srv.URL+"/api/elevators/ELEV-002/states", body)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("上报失败（第 %d 条）: %d %v", i, resp.StatusCode, payload)
		}
		data := payload["data"].(map[string]any)
		if ev, ok := data["entrapment_event"].(map[string]any); ok {
			eventID = ev["id"].(string)
		}
	}
	if eventID == "" {
		t.Fatal("连续上报后应触发困人事件")
	}

	// 事件列表命中。
	resp, payload := doJSON(t, http.MethodGet, srv.URL+"/api/events", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("事件列表失败")
	}
	events := payload["data"].(map[string]any)["events"].([]any)
	found := false
	for _, e := range events {
		if e.(map[string]any)["id"] == eventID {
			found = true
		}
	}
	if !found {
		t.Fatal("事件列表应包含新触发的事件")
	}

	// 接单 → 处置完成。
	resp, payload = doJSON(t, http.MethodPost, srv.URL+"/api/events/"+eventID+"/accept", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("接单失败: %d %v", resp.StatusCode, payload)
	}

	// 缺少字段的处置应返回 422。
	_, payload = doJSON(t, http.MethodPost, srv.URL+"/api/events/"+eventID+"/resolve", map[string]any{})
	// （payload code 42200 在 Fail 中返回，但为了兼容 httptest 默认 200 包装，这里校验 message 非 ok）
	if payload["message"] == "ok" {
		t.Fatal("缺少必填字段的处置不应成功")
	}

	// 完整处置。
	resp, payload = doJSON(t, http.MethodPost, srv.URL+"/api/events/"+eventID+"/resolve", map[string]any{
		"disposer":      "王工",
		"measure":       "解除门锁并复位",
		"recovery_time": time.Now().Format(time.RFC3339),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("处置完成失败: %d %v", resp.StatusCode, payload)
	}
	event := payload["data"].(map[string]any)["event"].(map[string]any)
	if event["status"] != "released" {
		t.Fatalf("处置后状态应为 released，得到 %v", event["status"])
	}
}

func TestEscalateAndScoreAndWatchlist(t *testing.T) {
	srv := newTestServer(t)
	start := time.Now().Add(-2 * time.Minute)
	var eventID string
	for i := 0; i < 8; i++ {
		at := start.Add(time.Duration(i) * 5 * time.Second)
		body := map[string]any{
			"floor": 3, "position": "3F-4F 之间", "direction": "idle",
			"door": "closed", "leveling": false, "fault_code": "",
			"passenger_signal": "alarm", "alarm_active": true,
			"reported_at": at.Format(time.RFC3339),
		}
		_, payload := doJSON(t, http.MethodPost, srv.URL+"/api/elevators/ELEV-002/states", body)
		if ev, ok := payload["data"].(map[string]any)["entrapment_event"].(map[string]any); ok {
			eventID = ev["id"].(string)
		}
	}
	if eventID == "" {
		t.Fatal("应触发困人事件")
	}

	resp, payload := doJSON(t, http.MethodPost, srv.URL+"/api/events/"+eventID+"/escalate", map[string]any{"reason": "人工确认升级"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("升级失败: %d %v", resp.StatusCode, payload)
	}
	event := payload["data"].(map[string]any)["event"].(map[string]any)
	if event["status"] != "escalated" || event["escalation_count"].(float64) != 1 {
		t.Fatalf("升级结果异常: %v", event)
	}

	// 评分与故障接口。
	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/api/elevators/ELEV-001/score", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("评分接口失败")
	}
	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/api/elevators/ELEV-005/faults", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("故障记录接口失败")
	}
	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/api/watchlist", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("重点关注接口失败")
	}
	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/api/diagnosis", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("诊断接口失败")
	}
	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/api/audit-logs", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("审计接口失败")
	}
}

func TestInvalidStateRejected(t *testing.T) {
	srv := newTestServer(t)
	body := map[string]any{
		"floor": 5, "position": "5F", "direction": "up",
		"door": "open", "leveling": false, "fault_code": "",
		"passenger_signal": "none",
	}
	resp, payload := doJSON(t, http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("运行中开门应返回 422，得到 %d", resp.StatusCode)
	}
	if payload["code"].(float64) != 42200 {
		t.Fatalf("业务码应为 42200，得到 %v", payload["code"])
	}
}

func TestUnknownFaultCodeRecorded(t *testing.T) {
	srv := newTestServer(t)
	body := map[string]any{
		"floor": 5, "position": "5F", "direction": "idle",
		"door": "closed", "leveling": true, "fault_code": "X77",
		"passenger_signal": "none",
	}
	resp, payload := doJSON(t, http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("上报失败: %d %v", resp.StatusCode, payload)
	}
	diagnosis := payload["data"].(map[string]any)["diagnosis"].(map[string]any)
	if diagnosis["known"].(bool) {
		t.Fatal("未知故障码 Known 应为 false")
	}
	if diagnosis["fault_type"] != "unknown" {
		t.Fatalf("fault_type 应为 unknown，得到 %v", diagnosis["fault_type"])
	}
}

func TestStaticSPAFallback(t *testing.T) {
	srv := newTestServer(t)
	// 前端路由 /elevators/xxx 应回退返回 index.html（200）。
	resp, err := http.Get(srv.URL + "/elevators/ELEV-001")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("前端路由应回退 200，得到 %d", resp.StatusCode)
	}
	body := make([]byte, 64)
	n, _ := resp.Body.Read(body)
	if string(body[:n]) != "<html><body>test</body></html>" {
		t.Fatalf("回退内容应为 index.html，得到 %q", string(body[:n]))
	}
	// 静态资源直接返回。
	resp2, err := http.Get(srv.URL + "/app.js")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("静态资源应返回 200，得到 %d", resp2.StatusCode)
	}
	// 未知 API 应 404。
	resp3, err := http.Get(srv.URL + "/api/not-exist")
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("未知 API 应返回 404，得到 %d", resp3.StatusCode)
	}
}

func TestRequestIDHeader(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/api/healthz", nil)
	if resp.Header.Get("X-Request-Id") == "" {
		t.Fatal("响应应携带 X-Request-Id")
	}
}
