package log

import (
	"encoding/json"
)

var F Formatter
var JF *JsonFomatter = &JsonFomatter{}

type Formatter interface {
	Format(s string) (string, error)
}

func Format(formatter Formatter, s string) (string, error) {
	return formatter.Format(s)
}

type JsonFomatter struct{}

func (jf *JsonFomatter) Format(s string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(s), &result)
	if err != nil {
		return "", err
	}
	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return "", err
	}
	return string(prettyJSON), nil
}
