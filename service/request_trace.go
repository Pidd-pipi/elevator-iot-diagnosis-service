package service

import "sync"

// RequestTrace 承载单个 HTTP 请求的作用域数据，供 handler → service 层
// 透传请求级信息（如 trace id）。
//
// 设计动机：业务审计日志（event.accept / event.resolve / event.alert 等）
// 在 service 层经 AuditService.Record / IngestService.auditRecord 写入，
// 而这些方法签名中并不携带 context。若为所有 service 方法追加 ctx 参数，
// 会破坏既有调用方与测试。RequestTrace 以「请求级临时容器」的形式，
// 在 handler 入口设置、在 service 写审计时读取，避免侵入式改签名。
//
// 生命周期：一次请求一个实例，由 handler 在 ServeHTTP 中创建并通过
// svc.Trace.Bind() 绑定；请求结束后不再被引用，由 GC 回收。
// 因此内部用 sync.Mutex 保证并发安全即可，无需清理逻辑。
//
// 注意：当 service 在非 HTTP 调用方执行（如定时扫描、种子数据、测试）
// 时，应使用 NewRequestTrace() 构造空实例，读取到的 request id 为空，
// 与「无请求则无 trace」的语义一致。
type RequestTrace struct {
	mu        sync.Mutex
	requestID string
}

// NewRequestTrace 构造一个空的请求作用域容器。
func NewRequestTrace() *RequestTrace { return &RequestTrace{} }

// SetRequestID 设置当前请求的 trace id。
func (t *RequestTrace) SetRequestID(id string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.requestID = id
	t.mu.Unlock()
}

// RequestID 返回当前请求的 trace id；未设置或容器为空时返回空串。
func (t *RequestTrace) RequestID() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requestID
}

// traceHolder 是 service 层各服务共享的「当前请求作用域容器」。
//
// 在非请求上下文（启动、扫描任务、种子数据、单测）下为 nil，
// 对应的 RequestID() 返回空串。Handler 在处理请求前调用 BindTrace
// 注入非空实例，使该请求期间 service 写入的业务审计日志能带上 trace id。
var traceHolder = (*RequestTrace)(nil)

// BindTrace 设置全局当前请求作用域容器，返回上一次绑定的容器以便恢复。
//
// 典型用法（handler）：
//
//	prev := BindTrace(NewRequestTrace())
//	defer BindTrace(prev)
//	// 处理请求……
//
// defer 恢复保证并发请求间互不串号。
func BindTrace(t *RequestTrace) *RequestTrace {
	traceMu.Lock()
	prev := traceHolder
	traceHolder = t
	traceMu.Unlock()
	return prev
}

var traceMu sync.Mutex

// CurrentTrace 返回当前绑定的请求作用域容器，可能为 nil。
func CurrentTrace() *RequestTrace {
	traceMu.Lock()
	defer traceMu.Unlock()
	return traceHolder
}
