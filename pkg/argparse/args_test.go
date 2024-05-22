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

package argparse

import (
	"os"
	"strings"
	"testing"
)

func TestAugment_ParseArgs(t *testing.T) {
	a := NewArgument()

	for _, tt := range []struct {
		name       string
		args       []string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Test with valid arguments",
			args: []string{
				"-type", "thrift", "-idl-path", "../../internal/test/example_service.thrift",
				"-method", "ExampleService/exampleMethod", "-data", "{\"Msg\": \"hello\"}",
				"-endpoint", "0.0.0.0:9999", "-verbose", "-transport", "TTHeader",
				"-meta", "temp=temp-value", "-meta-persistent", "logid=12345",
				"-meta-backward", "something",
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "Test with missing IDL path",
			args: []string{
				"-type", "protobuf", "-method", "example/hello",
				"-data", "world", "-endpoint", "0.0.0.0:9999", "-verbose",
				"-transport", "Framed",
			},
			wantErr:    true,
			wantErrMsg: "IDL path is required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate command-line arguments
			os.Args = append([]string{"cmd"}, tt.args...)
			err := a.ParseArgs()

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("ParseArgs() error message = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}
