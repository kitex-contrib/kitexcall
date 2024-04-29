package log

import (
	"fmt"
	"io"
	"os"
)

type Outputter interface {
	Success(data ...interface{}) error
	Fail(data ...interface{}) error
}

var Verbose bool

type Logger struct {
	Println func(w io.Writer, a ...interface{}) (n int, err error)
	Printf  func(w io.Writer, format string, a ...interface{}) (n int, err error)
}

var defaultLogger = Logger{
	Println: fmt.Fprintln,
	Printf:  fmt.Fprintf,
}

// SetDefaultLogger sets the default logger.
func SetDefaultLogger(l Logger) {
	defaultLogger = l
}

func (c *Logger) Success(data ...interface{}) error {
	defaultLogger.Println(os.Stdout, "\033[32mCall Status: Success\033[0m")
	defaultLogger.Println(os.Stdout, data...)
	return nil
}

func (c *Logger) Fail(data ...interface{}) error {
	defaultLogger.Println(os.Stderr, "\033[31mCall Status: Failed\033[0m")
	defaultLogger.Println(os.Stderr, data...)
	return nil
}

func (c *Logger) Info(data ...interface{}) {
	if Verbose {
		defaultLogger.Printf(os.Stdout, "[INFO]:%v\n", data...)
	}
}

func Info(data ...interface{}) {
	defaultLogger.Info(data...)
}

func Success(data ...interface{}) {
	defaultLogger.Success(data...)
}

func Fail(data ...interface{}) {
	defaultLogger.Fail(data...)
}

// For file output
type File struct{}

func (f *File) Success(data interface{}) error {
	return nil
}

func (f *File) Fail(data interface{}) error {
	return nil
}
