package domain

import "errors"

// 领域层统一定义的错误类型。httpapi 层根据错误类型映射 HTTP 状态码，
// 保证「领域错误 → 接口响应」的映射口径唯一。
var (
	// ErrNotFound 目标实体不存在。
	ErrNotFound = errors.New("资源不存在")
	// ErrConflict 业务冲突（例如同一电梯已存在未关闭的困人事件）。
	ErrConflict = errors.New("业务状态冲突")
	// ErrInvalidState 状态机非法迁移。
	ErrInvalidState = errors.New("非法状态迁移")
	// ErrValidation 参数校验失败。
	ErrValidation = errors.New("参数校验失败")
	// ErrDuplicate 重复提交（例如重复接单）。
	ErrDuplicate = errors.New("重复操作")
	// ErrInternal 内部错误兜底。
	ErrInternal = errors.New("内部错误")
)

// ValidationError 携带用户可读的校验失败详情。
type ValidationError struct {
	Field   string
	Message string
}

// Error 实现 error 接口。
func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError 构造一个带字段名的校验错误。
func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// IsValidationError 判断错误是否为校验失败。
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
