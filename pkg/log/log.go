package log

import (
	"fmt"
	"os"
)

type Logger struct {
	Formatter OutputFormatter
}

func Error(stage string, v ...interface{}) {
	// var msg string
	// switch stage {
	// case "ParseError":
	// 	msg = fmt.Sprintf("%s Error: %s\n", stage, fmt.Sprint(v...))
	// case "ServerError":
	// 	msg = fmt.Sprintf("%s Error: %s\n", stage, fmt.Sprint(v...))
	// default:
	// 	msg = fmt.Sprintf("%s Error: %s\n", stage, fmt.Sprint(v...))
	// }
	msg := fmt.Sprintf("%s Error: %s\n", stage, fmt.Sprint(v...))
	fmt.Fprint(os.Stderr, msg)
}
