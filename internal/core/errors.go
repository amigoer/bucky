package core

import (
	"context"
	"errors"
	"fmt"
)

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Provider  string `json:"provider,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Retryable bool   `json:"retryable"`
}

func (e *Error) Error() string { return e.Message }

func NewError(code, message string) *Error { return &Error{Code: code, Message: message} }

func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var result *Error
	if errors.As(err, &result) {
		return result
	}
	if errors.Is(err, context.Canceled) {
		return NewError("cancelled", "Operation cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: "timeout", Message: "Operation timed out", Retryable: true}
	}
	return NewError("internal", "Operation failed; check the local configuration and retry")
}

func Invalid(format string, args ...any) *Error {
	return NewError("invalid_argument", fmt.Sprintf(format, args...))
}
