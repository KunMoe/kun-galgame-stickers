package errors

import "fmt"

type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

const (
	CodeOK   = 0
	CodeAuth = 205
	CodeBiz  = 233
)

func ErrUnauthorized(msg string) *AppError {
	return New(CodeAuth, msg, 401)
}

func ErrBadRequest(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}

func ErrNotFound(msg string) *AppError {
	return New(CodeBiz, msg, 404)
}

func ErrInternal(msg string) *AppError {
	return New(CodeBiz, msg, 500)
}
