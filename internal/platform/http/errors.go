package http

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewValidationError(message string, err error) *AppError {
	return &AppError{
		Code:       "validation_error",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Code:       "unauthorized",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Err:        err,
	}
}

func NewForbiddenError(message string, err error) *AppError {
	return &AppError{
		Code:       "forbidden",
		Message:    message,
		StatusCode: http.StatusForbidden,
		Err:        err,
	}
}

func NewNotFoundError(message string, err error) *AppError {
	return &AppError{
		Code:       "not_found",
		Message:    message,
		StatusCode: http.StatusNotFound,
		Err:        err,
	}
}

func NewConflictError(message string, err error) *AppError {
	return &AppError{
		Code:       "conflict",
		Message:    message,
		StatusCode: http.StatusConflict,
		Err:        err,
	}
}

func NewInvalidStateError(message string, err error) *AppError {
	return &AppError{
		Code:       "invalid_state_transition",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

func NewInsufficientInventoryError(message string, err error) *AppError {
	return &AppError{
		Code:       "insufficient_inventory",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:       "internal_error",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if err == nil {
		return nil, false
	}
	if err, ok := err.(*AppError); ok {
		return err, true
	}
	return nil, false
}