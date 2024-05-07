/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"
	"io"
	"os"
)

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

func Success() {
	defaultLogger.Println(os.Stderr, "\033[32m[Status]: Success\033[0m")
}

func Fail() {
	defaultLogger.Println(os.Stderr, "\033[31m[Status]: Failed\033[0m")
}

func Println(v ...interface{}) {
	defaultLogger.Println(os.Stderr, v...)
}

func Printf(format string, v ...interface{}) {
	defaultLogger.Printf(os.Stderr, format, v...)
}

func Warn(v ...interface{}) {
	defaultLogger.Println(os.Stderr, v...)
}

func Warnf(format string, v ...interface{}) {
	defaultLogger.Printf(os.Stderr, format, v...)
}

func Info(v ...interface{}) {
	if Verbose {
		defaultLogger.Printf(os.Stderr, "[INFO]: %v\n", v...)
	}
}
