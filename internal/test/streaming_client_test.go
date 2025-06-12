/*
 * Copyright 2025 CloudWeGo Authors
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
	"net"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/server"
	testserver "github.com/kitex-contrib/kitexcall/internal/test/server"
	"github.com/kitex-contrib/kitexcall/internal/test/server/kitex_gen/streaming/streamingservice"
	"github.com/kitex-contrib/kitexcall/pkg/client"
	"github.com/kitex-contrib/kitexcall/pkg/config"
)

var (
	streamingServerHost = "127.0.0.1:9200"
	streamingServer     server.Server
	streamingThriftFile = "./streaming_service.thrift"
)

// InitStreamingServer initializes and starts the streaming server
func InitStreamingServer() {
	addr, err := net.ResolveTCPAddr("tcp", streamingServerHost)
	if err != nil {
		klog.Fatalf("Failed to resolve address: %v", err)
	}
	streamingServer = streamingservice.NewServer(new(testserver.StreamingServiceImpl),
		server.WithServiceAddr(addr))
	go func() {
		err := streamingServer.Run()
		if err != nil {
			klog.Fatalf("server stopped with error: %v", err)
		}
	}()
	WaitStreamingServerStart(streamingServerHost)
}

// WaitStreamingServerStart waits for the streaming server to start
func WaitStreamingServerStart(addr string) {
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			klog.Infof("server is up at %s", addr)
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
	panic("server failed to start")
}

func TestBidirectionalStreaming(t *testing.T) {
	InitStreamingServer()
	defer streamingServer.Stop()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{streamingServerHost},
		IDLPath:        streamingThriftFile,
		IDLServiceName: "StreamingService",
		Method:         "BidirectionalStream",
		Data:           "{\"Msg\": \"hello\"}",
		Transport:      config.GRPC,
		IsStreaming:    true,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from StreamingClient
	streamCli := cli.(*client.StreamingClient)
	if streamCli == nil {
		t.Fatalf("Stream client is nil")
	}

	// Verify streaming mode
	mt, err := streamCli.Generic.GetMethod(nil, conf.Method)
	if err != nil {
		t.Fatalf("Failed to get method info: %v", err)
	}
	if mt.StreamingMode != serviceinfo.StreamingBidirectional {
		t.Errorf("Expected bidirectional streaming mode, got %v", mt.StreamingMode)
	}

	// Execute the streaming call
	err = streamCli.Call()
	if err != nil {
		t.Fatalf("Streaming call failed: %v", err)
	}
}

func TestServerStreaming(t *testing.T) {
	InitStreamingServer()
	defer streamingServer.Stop()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{streamingServerHost},
		IDLPath:        streamingThriftFile,
		IDLServiceName: "StreamingService",
		Method:         "ServerStream",
		Data:           "{\"Msg\": \"hello\"}",
		Transport:      config.GRPC,
		IsStreaming:    true,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from StreamingClient
	streamCli := cli.(*client.StreamingClient)
	if streamCli == nil {
		t.Fatalf("Stream client is nil")
	}

	// Verify streaming mode
	mt, err := streamCli.Generic.GetMethod(nil, conf.Method)
	if err != nil {
		t.Fatalf("Failed to get method info: %v", err)
	}
	if mt.StreamingMode != serviceinfo.StreamingServer {
		t.Errorf("Expected server streaming mode, got %v", mt.StreamingMode)
	}
}

func TestClientStreaming(t *testing.T) {
	InitStreamingServer()
	defer streamingServer.Stop()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{streamingServerHost},
		IDLPath:        streamingThriftFile,
		IDLServiceName: "StreamingService",
		Method:         "ClientStream",
		Data:           "{\"Msg\": \"hello\"}",
		Transport:      config.GRPC,
		IsStreaming:    true,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from StreamingClient
	streamCli := cli.(*client.StreamingClient)
	if streamCli == nil {
		t.Fatalf("Stream client is nil")
	}

	// Verify streaming mode
	mt, err := streamCli.Generic.GetMethod(nil, conf.Method)
	if err != nil {
		t.Fatalf("Failed to get method info: %v", err)
	}
	if mt.StreamingMode != serviceinfo.StreamingClient {
		t.Errorf("Expected client streaming mode, got %v", mt.StreamingMode)
	}
}

func TestStreamingWithStdinInput(t *testing.T) {
	InitStreamingServer()
	defer streamingServer.Stop()

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
	testData := `{"Msg": "hello"}
{"Msg": "world"}`
	go func() {
		_, err := w.WriteString(testData + "\n")
		if err != nil {
			t.Errorf("Failed to write to pipe: %v", err)
		}
		w.Close()
	}()

	conf := &config.Config{
		Type:           config.Thrift,
		Endpoint:       []string{streamingServerHost},
		IDLPath:        streamingThriftFile,
		IDLServiceName: "StreamingService",
		Method:         "BidirectionalStream",
		Transport:      config.GRPC,
		IsStreaming:    true,
		MetaBackward:   true,
		Meta:           map[string]string{"temp": "temp-value"},
		MetaPersistent: map[string]string{"logid": "12345"},
	}

	cli, err := client.InvokeRPC(conf)
	if err != nil {
		t.Fatalf("InvokeRPC failed: %v", err)
	}

	// Get response from StreamingClient
	streamCli := cli.(*client.StreamingClient)
	if streamCli == nil {
		t.Fatalf("Stream client is nil")
	}

	// Verify streaming mode
	mt, err := streamCli.Generic.GetMethod(nil, conf.Method)
	if err != nil {
		t.Fatalf("Failed to get method info: %v", err)
	}
	if mt.StreamingMode != serviceinfo.StreamingBidirectional {
		t.Errorf("Expected bidirectional streaming mode, got %v", mt.StreamingMode)
	}
}
