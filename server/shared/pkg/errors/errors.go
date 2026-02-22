package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func New(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NewBadRequest(msg string, err error) *AppError   { return New(http.StatusBadRequest, msg, err) }
func NewUnauthorized(msg string, err error) *AppError { return New(http.StatusUnauthorized, msg, err) }
func NewForbidden(msg string, err error) *AppError    { return New(http.StatusForbidden, msg, err) }
func NewNotFound(msg string, err error) *AppError     { return New(http.StatusNotFound, msg, err) }

func NewInternal(msg string, err error) *AppError {
	if msg == "" {
		msg = "Internal Server Error"
	}
	return New(http.StatusInternalServerError, msg, err)
}
