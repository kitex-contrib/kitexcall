package log

import "fmt"

type OutputFormatter interface {
	Format() string
}

func FormatResult(formatter OutputFormatter) string {
	return formatter.Format()
}

type ErrorMessage struct {
	Message string
}

func (em *ErrorMessage) Format() string {
	return fmt.Sprintf("Error: %s\n", em.Message)
}
