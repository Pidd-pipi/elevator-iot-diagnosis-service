package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/service"
	"example.com/elevator-iot-diagnosis-service/store"
)

func newBug003Router(t *testing.T) (*httptest.Server, *store.Store) {
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
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log(1)")},
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, st
}

// 并发请求下，每个响应体都必须携带与响应头一致的 request_id（客户端不传 X-Request-Id）。
func TestBug003ResponseRequestIDPresent(t *testing.T) {
	srv, _ := newBug003Router(t)
	const n = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/healthz", nil)
			if err != nil {
				errCh <- "构造请求失败"
				return
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errCh <- "请求失败"
				return
			}
			defer resp.Body.Close()
			ridHeader := resp.Header.Get("X-Request-Id")
			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				errCh <- "响应解析失败"
				return
			}
			ridBody, _ := payload["request_id"].(string)
			if ridHeader == "" || ridBody == "" || ridBody != ridHeader {
				errCh <- fmt.Sprintf("request_id 未正确传播: header=%q body=%q", ridHeader, ridBody)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
}

// 并发写请求下，审计日志必须为每条请求记录一致的 request_id。
func TestBug003AuditLogCarriesRequestID(t *testing.T) {
	srv, st := newBug003Router(t)
	const n = 6
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(map[string]any{
				"floor": 3, "position": "3F", "direction": "idle", "door": "closed",
				"leveling": true, "passenger_signal": "none",
			})
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/elevators/ELEV-001/states", &buf)
			if err != nil {
				errCh <- "构造请求失败"
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errCh <- "请求失败"
				return
			}
			rid := resp.Header.Get("X-Request-Id")
			_ = resp.Body.Close()
			if rid == "" {
				errCh <- "写请求响应头缺少 X-Request-Id"
			}
			ok := false
			for _, l := range st.Audits.List(0) {
				if l.Action == "http.request" && l.RequestID == rid {
					ok = true
					break
				}
			}
			if !ok {
				errCh <- fmt.Sprintf("审计日志缺少 request_id=%q", rid)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
}
