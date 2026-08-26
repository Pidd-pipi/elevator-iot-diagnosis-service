package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/middleware"
	"example.com/elevator-iot-diagnosis-service/service"
)

// Router 组装完整 HTTP 路由：
//  1. REST API 路由（按方法 + 路径注册）；
//  2. go:embed 静态资源 + SPA 回退；
//  3. 中间件链：requestID → recoverer → auditLogger。
//
// webFS 为 main 包通过 go:embed all:web 注入的内嵌前端资源。
func Router(svc *service.Services, cfg *config.Config, logger *slog.Logger, webFS fs.FS) http.Handler {
	api := NewAPI(svc, cfg, logger)

	health := NewHealthHandler()
	elevators := NewElevatorHandler(svc)
	reports := NewReportHandler(svc)
	events := NewEventHandler(svc)
	diagnose := NewDiagnoseHandler(api)
	scores := NewScoreHandler(api)
	watchlist := NewWatchlistHandler(api)
	overview := NewOverviewHandler(api)
	audits := NewAuditHandler(api)

	mux := http.NewServeMux()

	// 健康检查与就绪检查。
	mux.HandleFunc("GET /healthz", health.Health)
	mux.HandleFunc("GET /readyz", health.Ready)
	mux.HandleFunc("GET /api/healthz", health.Health)

	// 电梯台账与评分。
	mux.HandleFunc("GET /api/elevators", elevators.List)
	mux.HandleFunc("GET /api/elevators/{id}", elevators.Get)
	mux.HandleFunc("GET /api/elevators/{id}/score", scores.Score)
	mux.HandleFunc("GET /api/elevators/{id}/faults", diagnose.Faults)

	// 状态上报（采集链路入口）。
	mux.HandleFunc("POST /api/elevators/{id}/states", reports.Ingest)

	// 困人事件与处置流转。
	mux.HandleFunc("GET /api/events", events.List)
	mux.HandleFunc("GET /api/events/{id}", events.Get)
	mux.HandleFunc("POST /api/events/{id}/accept", events.Accept)
	mux.HandleFunc("POST /api/events/{id}/resolve", events.Resolve)
	mux.HandleFunc("POST /api/events/{id}/escalate", events.Escalate)

	// 重点关注、故障诊断、总览、审计。
	mux.HandleFunc("GET /api/watchlist", watchlist.Watchlist)
	mux.HandleFunc("GET /api/diagnosis", diagnose.Rules)
	mux.HandleFunc("GET /api/overview", overview.Overview)
	mux.HandleFunc("GET /api/audit-logs", audits.List)

	// 静态资源与 SPA 路由回退。
	mux.Handle("/", staticHandler(webFS))

	// 中间件链（执行顺序由外到内）：
	// requestID → securityHeaders → auditLogger → recoverer → mux。
	// auditLogger 包在 recoverer 外层，确保 panic 被恢复后的 500 响应也能记录访问日志。
	var handler http.Handler = mux
	handler = middleware.Recoverer(logger)(handler)
	handler = middleware.AuditLogger(svc.Store, logger)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.RequestID(handler)
	return handler
}

// staticHandler 提供内嵌前端资源，并为前端路由回退 index.html。
func staticHandler(webFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		// 静态资源存在则直接返回。
		if _, err := fs.Stat(webFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// API 未匹配的路径直接 404。
		if strings.HasPrefix(path, "api/") {
			http.NotFound(w, r)
			return
		}
		// SPA 回退：前端路由（/elevators/xxx、/events 等）统一返回 index.html。
		data, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			http.Error(w, "index.html 缺失", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
