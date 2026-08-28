package domain

import (
	"errors"
	"fmt"
)

type Error struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Path    []string `json:"path,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func NewError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "internal"
}
