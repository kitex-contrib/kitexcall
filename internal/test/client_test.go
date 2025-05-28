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

package test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/gopkg/cloud/metainfo"
	dproto "github.com/cloudwego/dynamicgo/proto"

	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/server"
	"github.com/cloudwego/kitex/server/genericserver"
	"github.com/kitex-contrib/kitexcall/pkg/client"
	"github.com/kitex-contrib/kitexcall/pkg/config"
)

var (
	thriftGenericServer               server.Server
	pbGenericServer                   server.Server
	bizErrorGenericServer             server.Server
	thriftGenericServerWithImportPath server.Server
	thriftServerHost                  = "127.0.0.1:9919"
	pbServerHostPort                  = "127.0.0.1:9199"
	bizErrorServerHost                = "127.0.0.1:9109"
	pbFilePath                        = "./example_service.proto"
	thriftFilePath                    = "./example_service.thrift"
	example2ServerHost                = "127.0.0.1:9100"
	example2FilePath                  = "./example2.thrift"
)

func InitPbGenericServer() {
	dOpts := dproto.Options{}
	p, err := generic.NewPbFileProviderWithDynamicGo(pbFilePath, context.Background(), dOpts)
	if err != nil {
		panic(err)
	}

	g, err := generic.JSONPbGeneric(p)
	if err != nil {
		panic(err)
	}

	go func() {
		var opts []server.Option
		addr, _ := net.ResolveTCPAddr("tcp", pbServerHostPort)
		opts = append(opts, server.WithServiceAddr(addr))

		pbGenericServer = genericserver.NewServer(new(GenericServiceImpl), g, opts...)
		klog.Infof("Starting pb generic server on %s", addr.String())

		if err := pbGenericServer.Run(); err != nil {
			klog.Infof(err.Error())
		}
	}()

	WaitServerStart(pbServerHostPort)
}

func InitThriftGenericServer() {
	p, err := generic.NewThriftFileProvider(thriftFilePath)
	if err != nil {
		panic(err)
	}
	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		panic(err)
	}

	go func() {
		addr, _ := net.ResolveTCPAddr("tcp", thriftServerHost)
		klog.Infof("Starting thrift generic server on %s", addr.String())

		thriftGenericServer = genericserver.NewServer(new(GenericServiceImpl), g, server.WithServiceAddr(addr))

		if err := thriftGenericServer.Run(); err != nil {
			klog.Fatalf("Failed to run generic server: %v", err)
		}
	}()

	WaitServerStart(thriftServerHost)
}

func InitThriftGenericServerWithImportPath() {
	p, err := generic.NewThriftFileProvider(example2FilePath, "./idl")
	if err != nil {
		panic(err)
	}
	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		panic(err)
	}

	go func() {
		addr, _ := net.ResolveTCPAddr("tcp", example2ServerHost)
		klog.Infof("Starting Example2 service on %s", addr.String())

		thriftGenericServerWithImportPath = genericserver.NewServer(new(GenericServiceImpl), g, server.WithServiceAddr(addr))

		if err := thriftGenericServerWithImportPath.Run(); err != nil {
			klog.Fatalf("Failed to run Example2 service: %v", err)
		}
	}()

	WaitServerStart(example2ServerHost)
}

// WaitServerStart waits for server to start for at most 1 second
func WaitServerStart(addr string) {
	for begin := time.Now(); time.Since(begin) < time.Second; {
		if _, err := net.Dial("tcp", addr); err == nil {
			klog.Infof("server is up at %s", addr)
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
}

type GenericServiceImpl struct{}

func (g *GenericServiceImpl) GenericCall(ctx context.Context, method string, request interface{}) (response interface{}, err error) {
	temp, ok1 := metainfo.GetValue(ctx, "temp")
	if ok1 {
		klog.Info(temp)
	} else {
		klog.Warn("`temp` not exist in server-1 context")
	}

	logid, ok2 := metainfo.GetPersistentValue(ctx, "logid")
	if ok2 {
		klog.Info(logid)
	} else {
		klog.Warn("`logid` not exist in server-1 context")
	}

	ok := metainfo.SendBackwardValue(ctx, "something-from-server", time.Now().String())
	if !ok {
		return nil, errors.New("it looks like the protocol does not support transmitting meta information backward")
	}

	return "{\"Msg\": \"world\"}", nil
}

type BizErrorServiceImpl struct{}

func (g *BizErrorServiceImpl) GenericCall(ctx context.Context, method string, request interface{}) (response interface{}, err error) {
	err = kerrors.NewBizStatusError(404, "not found")
	return nil, err
}

func InitBizErrorGenericServer() {
	p, err := generic.NewThriftFileProvider(thriftFilePath)
	if err != nil {
		panic(err)
	}
	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		panic(err)
	}

	go func() {
		addr, _ := net.ResolveTCPAddr("tcp", bizErrorServerHost)
		klog.Infof("Starting thrift generic server on %s", addr.String())

		bizErrorGenericServer = genericserver.NewServer(new(BizErrorServiceImpl), g, server.WithServiceAddr(addr),
			server.WithMetaHandler(transmeta.ServerTTHeaderHandler))

		if err := bizErrorGenericServer.Run(); err != nil {
			klog.Fatalf("Failed to run generic server: %v", err)
		}
	}()

	WaitServerStart(bizErrorServerHost)
}

func TestThriftGenericServer_invokeRPC(t *testing.T) {
	InitThriftGenericServer()
	defer thriftGenericServer.Stop()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{thriftServerHost},
		IDLPath:        thriftFilePath,
		IDLServiceName: "GenericService",
		Method:         "ExampleMethod",
		Data:           "{\"Msg\": \"hello\"}",
		Transport:      config.TTHeader,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from BaseClient
	baseCli := cli.(*client.GenericClient)
	resp := baseCli.Resp
	if resp == nil {
		t.Fatalf("Response is nil")
	}

	expectedResponse := `{"Msg":"world","BaseResp":{"StatusCode":0,"StatusMessage":""}}`

	var serverData, expectedData interface{}
	json.Unmarshal([]byte(resp.(string)), &serverData)
	json.Unmarshal([]byte(expectedResponse), &expectedData)
	DeepEqual(t, serverData, expectedData)

	// MetaBackward
	if conf.MetaBackward {
		if res := cli.GetMetaBackward(); res == nil {
			t.Errorf("Expected meta backward not found in response")
		}
	}
}

func TestPbGenericServer_invokeRPC(t *testing.T) {
	InitPbGenericServer()
	defer pbGenericServer.Stop()

	conf := &config.Config{
		Type:           config.Protobuf,
		Endpoint:       []string{pbServerHostPort},
		IDLPath:        pbFilePath,
		IDLServiceName: "GenericService",
		Method:         "ExampleMethod",
		Data:           "{\"Msg\": \"hello\"}",
		Transport:      config.TTHeader,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from BaseClient
	baseCli := cli.(*client.GenericClient)
	resp := baseCli.Resp
	if resp == nil {
		t.Fatalf("Response is nil")
	}

	expectedResponse := `{"Msg":"world"}`

	var serverData, expectedData interface{}
	json.Unmarshal([]byte(resp.(string)), &serverData)
	json.Unmarshal([]byte(expectedResponse), &expectedData)
	DeepEqual(t, serverData, expectedData)

	// MetaBackward
	if conf.MetaBackward {
		if res := cli.GetMetaBackward(); res == nil {
			t.Errorf("Expected meta backward not found in response")
		}
	}
}

func TestHandleBizError(t *testing.T) {
	InitBizErrorGenericServer()
	defer bizErrorGenericServer.Stop()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{bizErrorServerHost},
		IDLPath:        thriftFilePath,
		IDLServiceName: "GenericService",
		Method:         "ExampleMethod",
		Transport:      config.TTHeader,
		BizError:       true,
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Verify that client is correctly returned
	if cli == nil {
		t.Errorf("Expected client, got nil")
	}
}

func TestExample2Service_withImportPath(t *testing.T) {
	InitThriftGenericServerWithImportPath()
	defer thriftGenericServerWithImportPath.Stop()

	currentPath, _ := os.Getwd()
	includePath := filepath.Join(currentPath, "idl")

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{example2ServerHost},
		IDLPath:        example2FilePath,
		IDLServiceName: "Example2Service",
		Method:         "Example2Method",
		Data:           "{\"Msg\": \"hello\", \"User\": {\"UserID\": \"123\", \"UserName\": \"Alice\", \"Age\": 25}}",
		Transport:      config.TTHeader,
		MetaBackward:   true,
		IncludePath:    []string{includePath},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from BaseClient
	baseCli := cli.(*client.GenericClient)
	resp := baseCli.Resp
	if resp == nil {
		t.Fatalf("Response is nil")
	}

	expectedResponse := `{"Msg":"world", "BaseResp": {"StatusCode": 0, "StatusMessage": ""}}`

	var serverData, expectedData interface{}
	json.Unmarshal([]byte(resp.(string)), &serverData)
	json.Unmarshal([]byte(expectedResponse), &expectedData)
	DeepEqual(t, serverData, expectedData)

	// MetaBackward
	if conf.MetaBackward {
		if res := cli.GetMetaBackward(); res == nil {
			t.Errorf("Expected meta backward not found in response")
		}
	}
}

func TestStdinInput(t *testing.T) {
	// Initialize and start the server
	InitThriftGenericServer()
	defer thriftGenericServer.Stop()

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Save original stdin and restore it after the test
	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()
	os.Stdin = r

	// Write test data to the pipe
	testData := `{"Msg": "hello"}`
	go func() {
		_, err := w.WriteString(testData + "\n")
		if err != nil {
			t.Errorf("Failed to write to pipe: %v", err)
		}
		w.Close()
	}()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{thriftServerHost},
		IDLPath:        thriftFilePath,
		IDLServiceName: "GenericService",
		Method:         "ExampleMethod",
		Transport:      config.TTHeader,
		// No Data field, will use stdin
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from BaseClient
	baseCli := cli.(*client.GenericClient)
	resp := baseCli.Resp
	if resp == nil {
		t.Fatalf("Response is nil")
	}

	expectedResponse := `{"Msg":"world","BaseResp":{"StatusCode":0,"StatusMessage":""}}`

	var serverData, expectedData interface{}
	json.Unmarshal([]byte(resp.(string)), &serverData)
	json.Unmarshal([]byte(expectedResponse), &expectedData)
	DeepEqual(t, serverData, expectedData)
}
