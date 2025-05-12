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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
	"github.com/kitex-contrib/kitexcall/pkg/versions"
)

type Argument struct {
	config.Config
	help    bool
	version bool
}

func NewArgument() *Argument {
	return &Argument{
		Config: config.Config{
			Meta:           make(map[string]string),
			MetaPersistent: make(map[string]string),
		},
	}
}

type StringList []string

func (s *StringList) String() string {
	return strings.Join(*s, ",")
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type KVMap map[string]string

func (m *KVMap) String() string {
	return ""
}

func (m *KVMap) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return errors.New(errors.ArgParseError, "Invalid format for key-value pair: %s", value)
	}
	(*m)[parts[0]] = parts[1]
	return nil
}

func (a *Argument) buildFlags() *flag.FlagSet {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	f.BoolVar(&a.help, "help", false, `Print usage instructions and exit.`)
	f.BoolVar(&a.help, "h", false, `Print usage instructions and exit.`)

	f.StringVar(&a.Type, "type", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'. Supports inference by IDL file type, default 'thrift'.")
	f.StringVar(&a.Type, "t", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'. Supports inference by IDL file type, default 'thrift'.(shorthand)")

	f.StringVar(&a.IDLPath, "idl-path", "", "Specify the path of IDL file.")
	f.StringVar(&a.IDLPath, "p", "", "Specify the path of IDL file. (shorthand)")

	f.StringVar(&a.Method, "method", "", "Specify the method name in the format `IDLServiceName/MethodName` or just `MethodName`, such as `GenericService/ExampleMethod`. Note that if server opened MultiService mode, IDLServiceName is needed and you should set `transport` flag with TTHeader or TTHeaderFramed")
	f.StringVar(&a.Method, "m", "", "Specify the method name. (shorthand)")

	f.StringVar(&a.File, "file", "", "Specify the file path of input. Must be in JSON format.")
	f.StringVar(&a.File, "f", "", "Specify the file path of input. Must be in JSON format. (shorthand)")

	f.StringVar(&a.Data, "data", "", "Specify the data to be sent as a JSON formatted string.")
	f.StringVar(&a.Data, "d", "", "Specify the data to be sent as a JSON formatted string. (shorthand)")

	f.Var((*StringList)(&a.Endpoint), "endpoint", "Specify the server endpoints. Can be repeated.")
	f.Var((*StringList)(&a.Endpoint), "e", "Specify the server endpoints. Can be repeated. (shorthand)")

	f.BoolVar(&a.Verbose, "verbose", false, "Enable verbose mode.")
	f.BoolVar(&a.Verbose, "v", false, "Enable verbose mode. (shorthand)")

	f.StringVar(&a.Transport, "transport", "", "Specify the transport type. Can be 'TTHeader', 'Framed', 'TTHeaderFramed'.")

	f.BoolVar(&a.BizError, "biz-error", false, "Enable client to handle biz error.")

	f.Var((*KVMap)(&a.Meta), "meta", "Specify the transient metainfo. Can be specified multiple times in key=value format.")
	f.Var((*KVMap)(&a.MetaPersistent), "meta-persistent", "Specify the persistent metainfo. Can be specified multiple times in key=value format.")

	f.BoolVar(&a.MetaBackward, "meta-backward", false, "Enable receiving backward metainfo from server.")

	f.BoolVar(&a.version, "version", false, "Show the version of kitexcall.")

	f.BoolVar(&a.Quiet, "quiet", false, "Enable only print rpc response.")

	f.Var((*StringList)(&a.IncludePath), "include-path", "Specify additional paths for imported IDL files. Can be repeated.")

	// Add streaming related flags
	f.BoolVar(&a.IsStreaming, "streaming", false, "Enable streaming mode. Will be auto-detected from IDL if not specified.")

	return f
}

func (a *Argument) PreJudge(f *flag.FlagSet) {
	if len(os.Args) == 1 {
		log.Println("Too few arguments. Try 'kitexcall -help' for more details.\n")
		os.Exit(1)
	}

	if a.help {
		Usage(f)
		os.Exit(0)
	}
}

func (a *Argument) ParseArgs() error {
	f := a.buildFlags()
	if err := f.Parse(os.Args[1:]); err != nil {
		return errors.New(errors.ArgParseError, "Flag parse error: %v", err)
	}

	a.PreJudge(f)

	if a.version {
		fmt.Fprintln(os.Stderr, versions.Version)
		os.Exit(0)
	}

	log.Verbose = a.Verbose
	log.Info("Parse args succeed")

	if err := a.ValidateArgs(); err != nil {
		return err
	}

	return nil
}

func (a *Argument) ValidateArgs() error {
	var err error
	if err = a.checkIDL(); err != nil {
		return err
	}

	if err = a.checkData(); err != nil {
		return err
	}

	if err = a.checkService(); err != nil {
		return err
	}

	if err = a.checkTransport(); err != nil {
		return err
	}

	log.Info("Args validate succeed")
	return nil
}

func (a *Argument) checkTransport() error {
	transport := strings.ToLower(a.Transport)
	switch transport {
	case strings.ToLower(config.TTHeader):
		a.Transport = config.TTHeader
	case strings.ToLower(config.Framed):
		a.Transport = config.Framed
	case strings.ToLower(config.TTHeaderFramed):
		a.Transport = config.TTHeaderFramed
	case strings.ToLower(config.GRPC):
		a.Transport = config.GRPC
	case "":
		if a.IsStreaming {
			// gRPC for streaming by default
			a.Transport = config.GRPC
		}
	default:
		return errors.New(errors.ArgParseError, "Transport type is invalid")
	}
	return nil
}

func (a *Argument) checkData() error {
	// Data's priority is higher than File
	if a.Data != "" {
		return nil
	}

	if a.File == "" {
		return errors.New(errors.ArgParseError, "At least one of Data or File must be specified")
	}

	if _, err := os.Stat(a.File); os.IsNotExist(err) {
		return errors.New(errors.ArgParseError, "Input file does not exist")
	}

	// Check file extension for valid formats
	if a.IsStreaming {
		// For streaming calls, allow both .json and .jsonl files
		if !strings.HasSuffix(a.File, ".json") && !strings.HasSuffix(a.File, ".jsonl") {
			return errors.New(errors.ArgParseError, "Input file must be in .json or .jsonl format")
		}
	} else {
		// For non-streaming calls, only allow .json files
		if !strings.HasSuffix(a.File, ".json") {
			return errors.New(errors.ArgParseError, "Input file must be in JSON format")
		}
	}
	return nil
}

func (a *Argument) checkService() error {
	// This can be modified when service discovery is supported.
	if len(a.Endpoint) == 0 {
		return errors.New(errors.ArgParseError, "At least one endpoint must be specified")
	}

	if a.Method == "" {
		return errors.New(errors.ArgParseError, "Method name is required")
	}

	// Split the method string into service and method if '/'
	parts := strings.Split(a.Method, "/")
	switch len(parts) {
	case 2: // If there is exactly one '/', it's treated as a separator
		a.IDLServiceName = parts[0]
		a.Method = parts[1]
	case 1: // If there is no '/', the whole string is treated as the method name
		a.Method = parts[0]
		a.IDLServiceName = ""
	default: // If there is more than one '/', it's an error
		return errors.New(errors.ArgParseError, "Method name must be in the format `ServiceName/MethodName` or just `MethodName`")
	}

	return nil
}

func (a *Argument) checkIDL() error {
	// Since server reflection is not available yet,
	// this is the only way to obtain IDL.
	if a.IDLPath == "" {
		return errors.New(errors.ArgParseError, "IDL path is required")
	}

	if _, err := os.Stat(a.IDLPath); os.IsNotExist(err) {
		return errors.New(errors.ArgParseError, "IDL file does not exist")
	}

	if a.IncludePath != nil {
		for _, path := range a.IncludePath {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return errors.New(errors.ArgParseError, "Include path does not exist")
			}
		}
	}

	switch a.Type {
	case "thrift", "protobuf":
	case "unknown":
		if typ, ok := a.guessIDLType(); ok {
			a.Type = typ
		} else {
			return errors.New(errors.ArgParseError, "Can not guess an IDL type from %q (unknown suffix), please specify with the '-type' flag.", a.IDLPath)
		}
	default:
		return errors.New(errors.ArgParseError, "Unsupported IDL type: %q", a.Type)
	}
	return nil
}

func (a *Argument) guessIDLType() (string, bool) {
	switch {
	case strings.HasSuffix(a.IDLPath, ".thrift"):
		return "thrift", true
	case strings.HasSuffix(a.IDLPath, ".proto"):
		return "protobuf", true
	}
	return "unknown", false
}

func (a *Argument) BuildConfig() *config.Config {
	return &config.Config{
		Method:         a.Method,
		Verbose:        a.Verbose,
		Type:           a.Type,
		IDLPath:        a.IDLPath,
		Endpoint:       a.Endpoint,
		IDLServiceName: a.IDLServiceName,
		Data:           a.Data,
		File:           a.File,
		Transport:      a.Transport,
		Meta:           a.Meta,
		MetaPersistent: a.MetaPersistent,
		MetaBackward:   a.MetaBackward,
		BizError:       a.BizError,
		Quiet:          a.Quiet,
		IncludePath:    a.IncludePath,
		// Add streaming configurations
		IsStreaming: a.IsStreaming,
	}
}

func Usage(f *flag.FlagSet) {
	f.Usage()
	fmt.Fprintln(os.Stderr, `
Examples:
	
	# Specify the type of IDL as Thrift, the path of IDL file, service name, server endpoint, method name, and data to be sent
	kitexcall -t thrift -idl-path /path/to/idl/file.thrift -e 0.0.0.0:9999 -m ServiceName/MethodName -d '{"Msg": "hello"}'
	# Or use the file as input:
	kitexcall -t thrift -idl-path /path/to/idl/file.thrift -e 0.0.0.0:9999 -m ServiceName/MethodName -f /path/to/input/file.json
	
	# Streaming examples:
	# Client streaming - with multiple messages from a JSONL file:
	kitexcall -t thrift -idl-path /path/to/idl/file.thrift -e 0.0.0.0:9999 -m ServiceName/MethodName -f /path/to/input/messages.jsonl --streaming
	
	# Server streaming - send a request and receive multiple responses:
	kitexcall -t thrift -idl-path /path/to/idl/file.thrift -e 0.0.0.0:9999 -m ServiceName/MethodName -d '{"Msg": "hello"}' --streaming
	
	# Bidirectional streaming - send multiple messages and receive multiple responses:
	kitexcall -t thrift -idl-path /path/to/idl/file.thrift -e 0.0.0.0:9999 -m ServiceName/MethodName -f /path/to/input/messages.jsonl --streaming
	`)
}
