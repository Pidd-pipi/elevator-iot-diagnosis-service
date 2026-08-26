package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/elevator-iot-diagnosis-service/store"
)

func TestRequestIDInjected(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFrom(r.Context()) == "" {
			t.Error("context 中缺少 trace id")
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := RequestID(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("响应头应包含 X-Request-Id")
	}
}

func TestRequestIDPassthrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFrom(r.Context()); got != "trace-abc" {
			t.Errorf("应透传外部 trace id，得到 %q", got)
		}
	})
	handler := RequestID(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "trace-abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-Id"); got != "trace-abc" {
		t.Errorf("响应头应透传 trace id，得到 %q", got)
	}
}

func TestRecovererCatchesPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := Recoverer(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic 应返回 500，得到 %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("错误响应体不应为空")
	}
}

func TestAuditLoggerRecordsWriteRequests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := store.NewStore()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	handler := AuditLogger(st, logger)(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/events/x/accept", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if st.Audits.Count() != 1 {
		t.Fatalf("写请求应写入审计日志，得到 %d 条", st.Audits.Count())
	}
	// 读请求不写审计。
	req2 := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if st.Audits.Count() != 1 {
		t.Fatalf("读请求不应写入审计日志，得到 %d 条", st.Audits.Count())
	}
}
