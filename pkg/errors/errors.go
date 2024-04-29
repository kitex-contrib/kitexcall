package errors

import (
	"fmt"
)

type Error struct {
	Type    ErrorType
	Message string
}

type ErrorType int

const (
	ArgParseError ErrorType = iota
	BizError
	ClientError
	ServerError
	OutputError
)

func (e *Error) Error() string {
	return fmt.Sprintf("error[%d]: %s", e.Type, e.Message)
}

func New(Type ErrorType, format string, args ...interface{}) *Error {
	return &Error{
		Type:    Type,
		Message: fmt.Sprintf(format, args...),
	}
}
