package errorsx

import (
	"net/http"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *AppError) Error() string {
	return e.Message
}

func ErrNotFound(msg string) *AppError {
	if msg == "" {
		msg = "resource not found"
	}
	return &AppError{
		Code:    "ERR_NOT_FOUND",
		Message: msg,
		Status:  http.StatusNotFound,
	}
}

func ErrInvalidInput(msg string) *AppError {
	if msg == "" {
		msg = "invalid input"
	}
	return &AppError{
		Code:    "ERR_INVALID_INPUT",
		Message: msg,
		Status:  http.StatusBadRequest,
	}
}

func ErrInternalError(msg string) error {
	if msg == "" {
		msg = "internal server error"
	}

	return &AppError{
		Code:    "ERR_INTERNAL",
		Message: msg,
		Status:  http.StatusInternalServerError,
	}
}
