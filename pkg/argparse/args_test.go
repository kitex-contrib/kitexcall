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

func TestStreamingArgs(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "test_file*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte(`{"test": "value"}`))
	if err != nil {
		t.Fatal(err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Create a test jsonl file for client/bidirectional streaming
	tmpJsonlFile, err := os.CreateTemp("", "test_file*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpJsonlFile.Name())
	_, err = tmpJsonlFile.Write([]byte(`{"test": "message1"}
{"test": "message2"}
{"test": "message3"}`))
	if err != nil {
		t.Fatal(err)
	}
	err = tmpJsonlFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "Valid client streaming with jsonl file",
			args: []string{
				"-idl-path", "test.thrift",
				"-e", "127.0.0.1:8000",
				"-m", "TestService/ClientMethod",
				"-f", tmpJsonlFile.Name(),
				"--streaming",
			},
			expectErr: false,
		},
		{
			name: "Valid server streaming with json file",
			args: []string{
				"-idl-path", "test.thrift",
				"-e", "127.0.0.1:8000",
				"-m", "TestService/ServerMethod",
				"-f", tmpFile.Name(),
				"--streaming",
			},
			expectErr: false,
		},
		{
			name: "Valid bidirectional streaming with jsonl file",
			args: []string{
				"-idl-path", "test.thrift",
				"-e", "127.0.0.1:8000",
				"-m", "TestService/Method",
				"-f", tmpJsonlFile.Name(),
				"--streaming",
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Save and restore os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set up test args
			os.Args = append([]string{"kitexcall"}, test.args...)

			// Run test
			arg := NewArgument()
			err := arg.ParseArgs()

			if test.expectErr && err == nil {
				t.Errorf("Expected error but got nil")
			}

			if !test.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// If streaming is enabled, verify that transport is set to GRPC
			if !test.expectErr && arg.IsStreaming {
				if arg.Transport != "GRPC" {
					t.Errorf("Expected transport to be GRPC for streaming call, but got %s", arg.Transport)
				}
			}
		})
	}
}
