package httpapi

import (
	"net/http"

	"example.com/elevator-iot-diagnosis-service/middleware"
	"example.com/elevator-iot-diagnosis-service/service"
)

// RequestScope 将请求级 trace id 注入 service 层的请求作用域容器。
//
// 背景：业务审计日志（event.accept / event.resolve / event.alert 等）在
// service 层写入，而 service 方法签名不携带 context。RequestScope 在请求
// 入口把 trace id 绑定到 service.RequestTrace，使这些业务审计也能带上
// 与网关一致的 trace id；请求结束（defer）恢复上一轮绑定，保证并发请求
// 互不串号。
//
// 必须位于 RequestID 中间件内层：trace id 来自 RequestID 注入的 context。
func RequestScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace := service.NewRequestTrace()
		trace.SetRequestID(middleware.RequestIDFrom(r.Context()))
		prev := service.BindTrace(trace)
		defer service.BindTrace(prev)
		next.ServeHTTP(w, r)
	})
}
