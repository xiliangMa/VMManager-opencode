package errors

import (
	"fmt"
	"net/http"
)

type ErrorCode int

const (
	ErrCodeSuccess ErrorCode = 0

	ErrCodeBadRequest      ErrorCode = 40001
	ErrCodeUnauthorized    ErrorCode = 40101
	ErrCodeForbidden       ErrorCode = 40301
	ErrCodeNotFound        ErrorCode = 40401
	ErrCodeConflict        ErrorCode = 40901
	ErrCodeTooManyRequests ErrorCode = 42901
	ErrCodeInternalError   ErrorCode = 50001

	ErrCodeValidation ErrorCode = 40002

	ErrCodeUserNotFound       ErrorCode = 40410
	ErrCodeUserExists         ErrorCode = 40910
	ErrCodeInvalidCredentials ErrorCode = 40102

	ErrCodeVMNotFound ErrorCode = 40420
	ErrCodeVMConflict ErrorCode = 40920

	ErrCodeTemplateNotFound ErrorCode = 40430

	ErrCodeQuotaExceeded ErrorCode = 40302

	ErrCodeDatabase ErrorCode = 50002
	ErrCodeLibvirt  ErrorCode = 50003
)

type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

func NewError(code ErrorCode, message string, details ...string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	err.StatusCode = MapCodeToStatus(code)
	return err
}

func MapCodeToStatus(code ErrorCode) int {
	switch code {
	case ErrCodeBadRequest, ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeUnauthorized, ErrCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrCodeForbidden, ErrCodeQuotaExceeded:
		return http.StatusForbidden
	case ErrCodeNotFound, ErrCodeUserNotFound, ErrCodeVMNotFound, ErrCodeTemplateNotFound:
		return http.StatusNotFound
	case ErrCodeConflict, ErrCodeUserExists, ErrCodeVMConflict:
		return http.StatusConflict
	case ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrCodeInternalError, ErrCodeDatabase, ErrCodeLibvirt:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

type Response struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
	Meta    any       `json:"meta,omitempty"`
	Details string    `json:"details,omitempty"`
}

func Success(data any) Response {
	return Response{
		Code:    ErrCodeSuccess,
		Message: "success",
		Data:    data,
	}
}

func SuccessWithMeta(data any, meta any) Response {
	return Response{
		Code:    ErrCodeSuccess,
		Message: "success",
		Data:    data,
		Meta:    meta,
	}
}

func Fail(err error) Response {
	if appErr, ok := err.(*AppError); ok {
		return Response{
			Code:    appErr.Code,
			Message: appErr.Message,
			Data:    nil,
			Details: appErr.Details,
		}
	}
	return Response{
		Code:    ErrCodeInternalError,
		Message: "internal error",
		Data:    nil,
	}
}

func FailWithCode(code ErrorCode, message string) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

func FailWithDetails(code ErrorCode, message string, details string) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    nil,
		Details: details,
	}
}
