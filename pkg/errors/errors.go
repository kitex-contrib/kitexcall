package errors

import (
	"fmt"
)

// KitexCallError 表示kitexcall工具的错误。
type Error struct {
	Type    ErrorType
	Message string
}

// ErrorType 表示错误的类型。
type ErrorType int

// 定义各种错误类型的常量。
const (
	ArgParseError ErrorType = iota // 命令行参数解析错误
	ClientError                    // 客户端错误
	ServerError                    // 服务器错误
	// 可以添加更多的错误类型...
)

// Error 实现error接口。
func (e *Error) Error() string {
	return fmt.Sprintf("error[%d]: %s", e.Type, e.Message)
}

// New 创建一个新的KitexCallError。
func New(Type ErrorType, format string, args ...interface{}) *Error {
	return &Error{
		Type:    Type,
		Message: fmt.Sprintf(format, args...),
	}
}
