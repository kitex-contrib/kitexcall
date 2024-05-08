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
	"encoding/json"
)

var Jf *JsonFormatter = &JsonFormatter{}
var Mf *MapFormatter = &MapFormatter{}

func FormatJson(s string) (string, error) {
	return Jf.Format(s)
}

func FormatMap(m map[string]string) (string, error) {
	return Mf.Format(m)
}

type JsonFormatter struct{}

func (jf *JsonFormatter) Format(s string) (string, error) {
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

type MapFormatter struct{}

func (mf *MapFormatter) Format(m map[string]string) (string, error) {
	result, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return "", err
	}
	return string(result), nil
}
