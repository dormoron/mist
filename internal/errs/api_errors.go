package errs

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorType 定义标准错误类型
type ErrorType string

const (
	// 错误类型常量
	ErrorTypeValidation    ErrorType = "VALIDATION_ERROR"    // 数据验证错误
	ErrorTypeAuth          ErrorType = "AUTH_ERROR"          // 认证错误
	ErrorTypePermission    ErrorType = "PERMISSION_ERROR"    // 权限错误
	ErrorTypeResource      ErrorType = "RESOURCE_ERROR"      // 资源未找到
	ErrorTypeInput         ErrorType = "INPUT_ERROR"         // 输入错误
	ErrorTypeInternal      ErrorType = "INTERNAL_ERROR"      // 内部服务器错误
	ErrorTypeUnavailable   ErrorType = "UNAVAILABLE_ERROR"   // 服务不可用
	ErrorTypeRateLimit     ErrorType = "RATE_LIMIT_ERROR"    // 限流错误
	ErrorTypeTimeout       ErrorType = "TIMEOUT_ERROR"       // 超时错误
	ErrorTypeUnprocessable ErrorType = "UNPROCESSABLE_ERROR" // 无法处理的实体
)

// APIError 统一API错误结构
type APIError struct {
	Type    ErrorType `json:"type"`              // 错误类型
	Code    int       `json:"code"`              // HTTP状态码
	Message string    `json:"message"`           // 用户友好的错误信息
	Details any       `json:"details,omitempty"` // 详细错误信息(可选)
}

// Error 实现error接口
func (e *APIError) Error() string {
	return e.Message
}

// ToJSON 将APIError转换为JSON格式
func (e *APIError) ToJSON() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return []byte(`{"type":"INTERNAL_ERROR","code":500,"message":"Error serializing error response"}`)
	}
	return data
}

// WithDetails 添加错误详情
func (e *APIError) WithDetails(details any) *APIError {
	e.Details = details
	return e
}

// 预定义错误 - 输入验证
func NewValidationError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeValidation,
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

// 预定义错误 - 认证失败
func NewAuthError(message string) *APIError {
	if message == "" {
		message = "Authentication failed"
	}
	return &APIError{
		Type:    ErrorTypeAuth,
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

// 预定义错误 - 权限不足
func NewPermissionError(message string) *APIError {
	if message == "" {
		message = "Permission denied"
	}
	return &APIError{
		Type:    ErrorTypePermission,
		Code:    http.StatusForbidden,
		Message: message,
	}
}

// 预定义错误 - 资源未找到
func NewResourceNotFoundError(resource string) *APIError {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	return &APIError{
		Type:    ErrorTypeResource,
		Code:    http.StatusNotFound,
		Message: message,
	}
}

// 预定义错误 - 请求超时
func NewTimeoutError(message string) *APIError {
	if message == "" {
		message = "Request timed out"
	}
	return &APIError{
		Type:    ErrorTypeTimeout,
		Code:    http.StatusRequestTimeout,
		Message: message,
	}
}

// 预定义错误 - 请求频率过高
func NewRateLimitError(message string) *APIError {
	if message == "" {
		message = "Too many requests"
	}
	return &APIError{
		Type:    ErrorTypeRateLimit,
		Code:    http.StatusTooManyRequests,
		Message: message,
	}
}

// 预定义错误 - 服务器内部错误
func NewInternalError(message string) *APIError {
	if message == "" {
		message = "Internal server error"
	}
	return &APIError{
		Type:    ErrorTypeInternal,
		Code:    http.StatusInternalServerError,
		Message: message,
	}
}

// 预定义错误 - 服务不可用
func NewServiceUnavailableError(message string) *APIError {
	if message == "" {
		message = "Service unavailable"
	}
	return &APIError{
		Type:    ErrorTypeUnavailable,
		Code:    http.StatusServiceUnavailable,
		Message: message,
	}
}

// 从错误类型创建错误
func NewErrorFromType(errorType ErrorType, message string) *APIError {
	switch errorType {
	case ErrorTypeValidation:
		return NewValidationError(message)
	case ErrorTypeAuth:
		return NewAuthError(message)
	case ErrorTypePermission:
		return NewPermissionError(message)
	case ErrorTypeResource:
		return NewResourceNotFoundError(message)
	case ErrorTypeTimeout:
		return NewTimeoutError(message)
	case ErrorTypeRateLimit:
		return NewRateLimitError(message)
	case ErrorTypeInternal:
		return NewInternalError(message)
	case ErrorTypeUnavailable:
		return NewServiceUnavailableError(message)
	default:
		return NewInternalError(message)
	}
}

// 从标准错误创建API错误
func WrapError(err error) *APIError {
	if err == nil {
		return nil
	}

	// 检查是否已经是APIError
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}

	// 返回通用内部错误
	return NewInternalError(err.Error())
}

// 从HTTP状态码创建错误
func NewErrorFromStatus(statusCode int, message string) *APIError {
	switch statusCode {
	case http.StatusBadRequest:
		return NewValidationError(message)
	case http.StatusUnauthorized:
		return NewAuthError(message)
	case http.StatusForbidden:
		return NewPermissionError(message)
	case http.StatusNotFound:
		return NewResourceNotFoundError(message)
	case http.StatusRequestTimeout:
		return NewTimeoutError(message)
	case http.StatusTooManyRequests:
		return NewRateLimitError(message)
	case http.StatusInternalServerError:
		return NewInternalError(message)
	case http.StatusServiceUnavailable:
		return NewServiceUnavailableError(message)
	default:
		if statusCode >= 500 {
			return NewInternalError(message)
		}
		return &APIError{
			Type:    ErrorTypeInput,
			Code:    statusCode,
			Message: message,
		}
	}
}
