// Package httpapi 提供 REST API 与静态资源路由。
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"example.com/elevator-iot-diagnosis-service/domain"
	"example.com/elevator-iot-diagnosis-service/middleware"
)

// errorLogger 用于记录命中 mapError 默认分支（即未被识别为领域错误的
// 500 内部错误）的具体原因。由 Router 装配时通过 setErrorLogger 注入；
// 未注入时静默跳过日志，避免影响不依赖路由装配的纯单元测试。
var errorLogger *slog.Logger

// setErrorLogger 注入错误日志记录器，供 Fail 在 500 路径落盘根因。
func setErrorLogger(l *slog.Logger) { errorLogger = l }

// Response 统一响应格式：code=0 表示成功，非 0 表示业务错误码。
type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Data      any    `json:"data,omitempty"`
}

// OK 写入成功响应。
func OK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, r, http.StatusOK, Response{
		Code:      0,
		Message:   "ok",
		RequestID: middleware.RequestIDFrom(r.Context()),
		Data:      data,
	})
}

// Created 写入 201 成功响应。
func Created(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, r, http.StatusCreated, Response{
		Code:      0,
		Message:   "ok",
		RequestID: middleware.RequestIDFrom(r.Context()),
		Data:      data,
	})
}

// Fail 按领域错误类型映射 HTTP 状态码并写入统一错误响应。
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	status, code, msg := mapError(err)
	// 命中默认分支意味着错误未被识别为任一已知领域错误，返回 500。
	// 此时根因对排障至关重要，务必落盘日志（含请求标识），避免「500 但日志无痕」。
	if status == http.StatusInternalServerError && errorLogger != nil {
		errorLogger.Error("unmapped error -> 500",
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", middleware.RequestIDFrom(r.Context()),
			"err", err,
		)
	}
	writeJSON(w, r, status, Response{
		Code:      code,
		Message:   msg,
		RequestID: middleware.RequestIDFrom(r.Context()),
	})
}

// mapError 将领域错误映射为 (HTTP状态码, 业务码, 提示信息)。
func mapError(err error) (int, int, string) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		return http.StatusUnprocessableEntity, 42200, err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, 40400, err.Error()
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, 40900, err.Error()
	case errors.Is(err, domain.ErrDuplicate):
		return http.StatusConflict, 40901, err.Error()
	case errors.Is(err, domain.ErrInvalidState):
		return http.StatusConflict, 40902, err.Error()
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity, 42201, err.Error()
	default:
		return http.StatusInternalServerError, 50000, fmt.Sprintf("内部错误: %v", err)
	}
}

// writeJSON 写入 JSON 响应。API 响应不缓存，统一设置 no-store。
func writeJSON(w http.ResponseWriter, r *http.Request, status int, body Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if rid := middleware.RequestIDFrom(r.Context()); rid != "" {
		w.Header().Set("X-Request-Id", rid)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
