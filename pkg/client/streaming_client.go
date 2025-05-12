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

package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt/streamcall"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

// StreamingClient wraps the generic streaming client functionality
// It supports all types of streaming RPC: client-streaming, server-streaming, and bidirectional-streaming
type StreamingClient struct {
	Client       genericclient.Client // Basic generic client interface
	Generic      generic.Generic      // Generic serialization/deserialization provider
	Provider     interface{}          // IDL Provider
	Conf         *config.Config       // Configuration for the client
	ClientOpts   []client.Option      // Client options for connection
	CallOptions  []streamcall.Option  // Call options for streaming RPC
	Req          interface{}          // Request data
	Resp         interface{}          // Response data
	StreamCli    interface{}          // Different stream client interfaces based on stream type
	MetaBackward map[string]string    // Metadata received from the server
}

// NewStreamingClient creates a new StreamingClient
func NewStreamingClient() *StreamingClient {
	return &StreamingClient{
		CallOptions: []streamcall.Option{},
	}
}

// Init initializes the StreamingClient with the provided configuration
// It creates a generic StreamClient that can be used for all streaming types
// The specific streaming client (client-streaming, server-streaming, bidirectional)
// will be created later when the RPC is actually invoked based on StreamType.
func (c *StreamingClient) Init(conf *config.Config) error {
	var g generic.Generic
	var err error

	// Initialize the appropriate generic provider based on IDL type
	if conf.Type == config.Thrift {
		p, err := generic.NewThriftFileProvider(conf.IDLPath, conf.IncludePath...)
		if err != nil {
			return errors.New(errors.ClientError, "Failed to create Thrift provider: %v", err)
		}
		c.Provider = p
		g, err = generic.JSONThriftGeneric(p)
		if err != nil {
			return errors.New(errors.ClientError, "Failed to create Thrift generic: %v", err)
		}
	} else if conf.Type == config.Protobuf {
		dOpts := proto.Options{}
		p, err := generic.NewPbFileProviderWithDynamicGo(conf.IDLPath, context.Background(), dOpts, conf.IncludePath...)
		if err != nil {
			return errors.New(errors.ClientError, "Failed to create Protobuf provider: %v", err)
		}
		c.Provider = p
		g, err = generic.JSONPbGeneric(p)
		if err != nil {
			return errors.New(errors.ClientError, "Failed to create Protobuf generic: %v", err)
		}
	} else {
		return errors.New(errors.ClientError, "Unsupported IDL type: %s", conf.Type)
	}

	// Store generic and config
	c.Generic = g
	c.Conf = conf

	// Build client options
	if err := c.BuildClientOptions(); err != nil {
		return err
	}

	// Initialize the streaming client
	cli, err := genericclient.NewStreamingClient(conf.IDLServiceName, c.Generic, c.ClientOpts...)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create streaming client: %v", err)
	}
	c.Client = cli

	// Initialize MetaBackward map if we'll need it
	if conf.MetaBackward {
		c.MetaBackward = make(map[string]string)
	}

	return nil
}

// BuildClientOptions builds the client options for the streaming client
func (c *StreamingClient) BuildClientOptions() error {
	var opts []client.Option
	opts = append(opts, client.WithHostPorts(c.Conf.Endpoint...))

	// Set transport protocol based on configuration
	switch c.Conf.Transport {
	case config.GRPC:
		opts = append(opts, client.WithTransportProtocol(transport.GRPC))
	case "":
		// For streaming, we always need to use gRPC transport protocol
		opts = append(opts, client.WithTransportProtocol(transport.GRPC))
	default:
		// For streaming, we always need to use gRPC transport protocol
		log.Warnf("Streaming calls require GRPC transport. Your specified transport '%s' will be ignored.", c.Conf.Transport)
		opts = append(opts, client.WithTransportProtocol(transport.GRPC))
	}

	// Add metainfo handler for gRPC when needed
	if c.Conf.Meta != nil || c.Conf.MetaPersistent != nil || c.Conf.MetaBackward {
		opts = append(opts, client.WithMetaHandler(transmeta.ClientHTTP2Handler))
	}

	c.ClientOpts = opts
	return nil
}

// BuildRequest builds the request data from command line arguments or file
func (c *StreamingClient) BuildRequest() error {
	if c.Conf.Data != "" {
		// 使用命令行参数提供的数据
		c.Req = []string{c.Conf.Data}
	} else if c.Conf.File != "" {
		// 从文件读取请求数据
		if strings.HasSuffix(c.Conf.File, ".jsonl") {
			// 读取多条消息，每行一条
			messages, err := readMessagesFromFile(c.Conf.File)
			if err != nil {
				return errors.New(errors.ClientError, "failed to read messages from file: %v", err)
			}
			c.Req = messages
		} else {
			// 读取单条消息
			data, err := os.ReadFile(c.Conf.File)
			if err != nil {
				return errors.New(errors.ClientError, "failed to read file: %v", err)
			}
			c.Req = []string{string(data)}
		}
	}
	return nil
}

// Call executes the streaming RPC call
func (c *StreamingClient) Call() error {
	// Create context with streaming call options
	ctx, err := c.BuildCallOptions()
	if err != nil {
		return err
	}

	// 使用 Generic.GetMethod 获取方法信息
	mt, err := c.Generic.GetMethod(nil, c.Conf.Method)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to get method info: %v", err)
	}

	// Handle the streaming RPC based on the streaming mode
	switch mt.StreamingMode {
	case serviceinfo.StreamingBidirectional:
		return c.handleBidirectionalStreaming(ctx)
	case serviceinfo.StreamingServer:
		return c.handleServerStreaming(ctx)
	case serviceinfo.StreamingClient:
		return c.handleClientStreaming(ctx)
	case serviceinfo.StreamingNone:
		return c.handleUnaryStreaming(ctx)
	default:
		return errors.New(errors.ClientError, "Unsupported streaming mode: %v", mt.StreamingMode)
	}
}

// handleClientStreaming handles client streaming calls
// This method:
// 1. Creates a client-streaming-specific client using the generic StreamClient
// 2. Reads messages from file (supporting both .json and .jsonl formats) or from command line data
// 3. Sends all messages to the server
// 4. Receives a single response after all messages are sent
func (c *StreamingClient) handleClientStreaming(ctx context.Context) error {
	// Create client streaming client
	streamCli, err := genericclient.NewClientStreaming(ctx, c.Client, c.Conf.Method)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create client streaming client: %v", err)
	}
	c.StreamCli = streamCli

	// Get messages using the shared BuildRequest method
	if err := c.BuildRequest(); err != nil {
		return err
	}

	// Access messages from c.Req
	messages := c.Req.([]string)

	// Send all messages
	for i, msg := range messages {
		if !c.Conf.Quiet {
			log.Info("Sending message %d: %s", i+1, msg)
		}
		if err := streamCli.Send(msg); err != nil {
			return errors.New(errors.ClientError, "Failed to send message: %v", err)
		}
	}

	// Close and receive response
	resp, err := streamCli.CloseAndRecv()
	if err != nil {
		return errors.New(errors.ServerError, "Failed to receive response: %v", err)
	}

	// Process response
	strResp, ok := resp.(string)
	if !ok {
		return errors.New(errors.OutputError, "Unexpected response type: %T", resp)
	}

	// Store response data
	c.Resp = strResp

	// Format and output the response
	formattedResp, err := log.FormatJson(strResp)
	if err != nil {
		return errors.New(errors.OutputError, "Failed to format response: %v", err)
	}

	if !c.Conf.Quiet {
		log.Success()
	}
	fmt.Print(formattedResp)

	return nil
}

// handleServerStreaming handles server streaming calls
func (c *StreamingClient) handleServerStreaming(ctx context.Context) error {
	// Get request data using the shared BuildRequest method
	if err := c.BuildRequest(); err != nil {
		return err
	}

	// For server streaming, we only use the first message
	var reqData string
	messages := c.Req.([]string)
	if len(messages) > 0 {
		reqData = messages[0]
	}

	// Create server streaming client and send the initial request
	streamCli, err := genericclient.NewServerStreaming(ctx, c.Client, c.Conf.Method, reqData)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create server streaming client: %v", err)
	}
	c.StreamCli = streamCli

	// Receive response stream
	responseCount := 0
	for {
		resp, err := streamCli.Recv()
		if err == io.EOF {
			// End of stream
			if !c.Conf.Quiet {
				log.Info("Stream closed after receiving %d responses", responseCount)
			}
			break
		} else if err != nil {
			return errors.New(errors.ServerError, "Failed to receive response: %v", err)
		}

		// Process response
		strResp, ok := resp.(string)
		if !ok {
			return errors.New(errors.OutputError, "Unexpected response type: %T", resp)
		}

		// Format and output the response
		formattedResp, err := log.FormatJson(strResp)
		if err != nil {
			return errors.New(errors.OutputError, "Failed to format response: %v", err)
		}

		if responseCount > 0 {
			// Add a separator between responses
			fmt.Println("---")
		}

		if !c.Conf.Quiet {
			log.Success()
		}
		fmt.Print(formattedResp)

		responseCount++
	}

	return nil
}

// handleBidirectionalStreaming handles bidirectional streaming calls
func (c *StreamingClient) handleBidirectionalStreaming(ctx context.Context) error {
	// Create bidirectional streaming client
	streamCli, err := genericclient.NewBidirectionalStreaming(ctx, c.Client, c.Conf.Method)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create bidirectional streaming client: %v", err)
	}
	c.StreamCli = streamCli

	// Get messages using the shared BuildRequest method
	if err := c.BuildRequest(); err != nil {
		return err
	}

	messages := c.Req.([]string)

	// Use wait group to coordinate send and receive goroutines
	wg := &sync.WaitGroup{}
	wg.Add(2)
	errCh := make(chan error, 2)

	// Send goroutine
	go func() {
		defer wg.Done()
		defer func() {
			if err := recover(); err != nil {
				errCh <- fmt.Errorf("panic in send goroutine: %v", err)
			}
		}()
		defer streamCli.Close() // Close the send side when done

		for i, msg := range messages {
			if !c.Conf.Quiet {
				log.Info("Sending message %d: %s", i+1, msg)
			}
			if err := streamCli.Send(msg); err != nil {
				errCh <- errors.New(errors.ClientError, "Failed to send message: %v", err)
				return
			}
		}
	}()

	// Receive goroutine
	go func() {
		defer wg.Done()
		defer func() {
			if err := recover(); err != nil {
				errCh <- fmt.Errorf("panic in receive goroutine: %v", err)
			}
		}()

		responseCount := 0
		for {
			resp, err := streamCli.Recv()
			if err == io.EOF {
				// End of stream
				if !c.Conf.Quiet {
					log.Info("Stream closed after receiving %d responses", responseCount)
				}
				break
			} else if err != nil {
				errCh <- errors.New(errors.ServerError, "Failed to receive response: %v", err)
				return
			}

			// Process response
			strResp, ok := resp.(string)
			if !ok {
				errCh <- errors.New(errors.OutputError, "Unexpected response type: %T", resp)
				return
			}

			// Format and output the response
			formattedResp, err := log.FormatJson(strResp)
			if err != nil {
				errCh <- errors.New(errors.OutputError, "Failed to format response: %v", err)
				return
			}

			if responseCount > 0 {
				// Add a separator between responses
				fmt.Println("---")
			}

			if !c.Conf.Quiet {
				log.Success()
			}
			fmt.Print(formattedResp)

			responseCount++
		}
	}()

	// Wait for both goroutines to finish
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Check for errors
	for err := range errCh {
		return err
	}

	return nil
}

// handleUnaryStreaming handles unary streaming calls (gRPC unary over streaming transport)
func (c *StreamingClient) handleUnaryStreaming(ctx context.Context) error {
	// Get request data using the shared BuildRequest method
	if err := c.BuildRequest(); err != nil {
		return err
	}

	// For unary streaming, we only use the first message
	var reqData string
	messages := c.Req.([]string)
	if len(messages) > 0 {
		reqData = messages[0]
	}

	// For unary streaming, we use the regular GenericCall
	// We don't need to convert streamcall.Options to callopt.Options as they are only for streaming RPCs
	resp, err := c.Client.GenericCall(ctx, c.Conf.Method, reqData)
	if err != nil {
		// Handle Biz error
		bizErr, isBizErr := kerrors.FromBizStatusError(err)
		if isBizErr {
			if err := c.handleBizError(bizErr); err != nil {
				return errors.New(errors.OutputError, "BizError parse error: %v", err)
			}
			return nil
		}
		return errors.New(errors.ServerError, "RPC call failed: %v", err)
	}

	// Process response
	strResp, ok := resp.(string)
	if !ok {
		return errors.New(errors.OutputError, "Unexpected response type: %T", resp)
	}

	// Format and output the response
	formattedResp, err := log.FormatJson(strResp)
	if err != nil {
		return errors.New(errors.OutputError, "Failed to format response: %v", err)
	}

	if !c.Conf.Quiet {
		log.Success()
	}
	fmt.Print(formattedResp)

	return nil
}

// handleBizError handles business errors
func (c *StreamingClient) handleBizError(bizErr kerrors.BizStatusErrorIface) error {
	var r string
	var err error
	if len(bizErr.BizExtra()) != 0 {
		r, err = log.FormatMap(bizErr.BizExtra())
		if err != nil {
			return err
		}
	}
	log.Success()
	log.Println("\033[32mReceived metainfo from server: \033[0m")
	log.Printf("BizStatusCode: %d\n", bizErr.BizStatusCode())
	log.Printf("BizMessage: %s\n", bizErr.BizMessage())

	if len(bizErr.BizExtra()) != 0 {
		log.Println("BizExtra:")
		log.Println(r)
	}
	return nil
}

// readMessagesFromFile reads JSON messages from a file, one per line
func readMessagesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			messages = append(messages, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetResponse returns the response
func (c *StreamingClient) GetResponse() interface{} {
	return c.Resp
}

// GetMetaBackward returns the metadata received from the server
func (c *StreamingClient) GetMetaBackward() map[string]string {
	if c.MetaBackward == nil {
		return make(map[string]string)
	}
	return c.MetaBackward
}

// Output handles the output for streaming responses
func (c *StreamingClient) Output() error {
	// Output is handled in the specific stream handling methods
	return nil
}

// HandleBizError handles business errors for streaming RPCs
func (c *StreamingClient) HandleBizError(bizErr kerrors.BizStatusErrorIface) error {
	return c.handleBizError(bizErr)
}

// BuildCallOptions builds the context with streaming call options
func (c *StreamingClient) BuildCallOptions() (context.Context, error) {
	ctx := context.Background()
	var opts []streamcall.Option

	// Handle metadata for gRPC streaming
	if c.Conf.Meta != nil {
		for k, v := range c.Conf.Meta {
			ctx = metainfo.WithValue(ctx, k, v)
		}
	}

	if c.Conf.MetaPersistent != nil {
		for k, v := range c.Conf.MetaPersistent {
			ctx = metainfo.WithPersistentValue(ctx, k, v)
		}
	}

	if c.Conf.MetaBackward {
		// must mark the context to receive backward meta information
		ctx = metainfo.WithBackwardValues(ctx)
	}

	c.CallOptions = opts
	return ctx, nil
}
