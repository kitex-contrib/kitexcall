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

package config

// Define constants for supported IDL types.
const (
	Unknown  string = ""
	Thrift   string = "thrift"
	Protobuf string = "protobuf"
	// Reserved for Server Streaming
)

const (
	TTHeader       string = "TTHeader"
	Framed         string = "Framed"
	TTHeaderFramed string = "TTHeaderFramed"
)

// We provide a general configuration
// so that it can be utilized by others apart from kitexcall.
type Config struct {
	Verbose        bool
	Type           string
	IDLPath        string
	Endpoint       []string
	IDLServiceName string
	Method         string
	Data           string
	Transport      string
	File           string // The file path of Input
	Meta           map[string]string
	MetaPersistent map[string]string
	MetaBackward   bool
	BizError       bool
}

type ConfigBuilder interface {
	BuildConfig() Config
}
