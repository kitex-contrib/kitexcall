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

package client

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

// StreamingClient implements the Client interface for streaming calls
type StreamingClient struct {
	BaseClient
}

// NewStreamingClient creates a new StreamingClient
func NewStreamingClient() *StreamingClient {
	return &StreamingClient{}
}

// Init initializes the StreamingClient
func (c *StreamingClient) Init(conf *config.Config) error {
	// Initialize base client
	if err := c.BaseClient.Init(conf); err != nil {
		return err
	}

	// Create streaming client
	cli, err := genericclient.NewStreamingClient(conf.IDLServiceName, c.Generic, c.ClientOpts...)
	if err != nil {
		return err
	}
	c.Client = cli

	return nil
}

// Call executes the streaming RPC call
func (c *StreamingClient) Call() error {
	// Create context with streaming call options
	ctx, err := c.buildCallOptions()
	if err != nil {
		return err
	}

	// Get method info
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
		return c.handlePingPong(ctx)
	default:
		return errors.New(errors.ClientError, "Unsupported streaming mode: %v", mt.StreamingMode)
	}
}

// buildCallOptions builds the context with streaming call options
func (c *StreamingClient) buildCallOptions() (context.Context, error) {
	ctx := context.Background()

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

	return ctx, nil
}

// handleClientStreaming handles client streaming calls
func (c *StreamingClient) handleClientStreaming(ctx context.Context) error {
	// Create a context that will be cancelled on Ctrl+C
	ctx = c.stream.WithInterruptHandler(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create streaming client
	streamCli, err := genericclient.NewClientStreaming(ctx, c.Client, c.Conf.Method)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create client streaming client: %v", err)
	}

	// Send messages
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := c.stream.Recv()
			if err != nil {
				if err == io.EOF {
					if !c.Conf.Quiet {
						log.Info("Received EOF, closing stream...")
					}
					// Close the stream and get response
					resp, err := streamCli.CloseAndRecv()
					if err != nil {
						return errors.New(errors.ServerError, "Failed to receive response: %v", err)
					}
					return c.stream.Send(resp)
				}
				return errors.New(errors.ClientError, "Failed to read message: %v", err)
			}

			if err := streamCli.Send(msg); err != nil {
				return errors.New(errors.ClientError, "Failed to send message: %v", err)
			}
		}
	}
}

// handleServerStreaming handles server streaming calls
func (c *StreamingClient) handleServerStreaming(ctx context.Context) error {
	// Create a context that will be cancelled on Ctrl+C
	ctx = c.stream.WithInterruptHandler(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get request data from ioStream
	reqData, err := c.stream.Recv()
	if err != nil && err != io.EOF {
		return errors.New(errors.ClientError, "Failed to read request: %v", err)
	}
	c.Req = reqData

	// Create streaming client with initial request
	streamCli, err := genericclient.NewServerStreaming(ctx, c.Client, c.Conf.Method, c.Req)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create server streaming client: %v", err)
	}

	// Receive response stream
	responseCount := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			resp, err := streamCli.Recv()
			if err != nil {
				if err == io.EOF {
					// Server ended the stream
					if !c.Conf.Quiet {
						log.Info("Stream closed by server after receiving %d responses", responseCount)
					}
					return nil
				}
				return errors.New(errors.ServerError, "Failed to receive response: %v", err)
			}

			if err := c.stream.Send(resp); err != nil {
				return errors.New(errors.OutputError, "Failed to send response: %v", err)
			}

			responseCount++
		}
	}
}

// handleBidirectionalStreaming handles bidirectional streaming calls
func (c *StreamingClient) handleBidirectionalStreaming(ctx context.Context) error {
	// Create a context that will be cancelled on Ctrl+C
	ctx = c.stream.WithInterruptHandler(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create streaming client
	streamCli, err := genericclient.NewBidirectionalStreaming(ctx, c.Client, c.Conf.Method)
	if err != nil {
		return errors.New(errors.ClientError, "Failed to create bidirectional streaming client: %v", err)
	}

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
		defer func() {
			if !c.Conf.Quiet {
				log.Info("Closing send stream...")
			}
			streamCli.Close() // Close the send side when done
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := c.stream.Recv()
				if err != nil {
					if err == io.EOF {
						// In interactive mode, EOF means user wants to stop the stream
						if c.Conf.File == "" {
							if !c.Conf.Quiet {
								log.Info("Received EOF from user input, closing send stream...")
							}
						}
						return
					}
					errCh <- errors.New(errors.ClientError, "Failed to read message: %v", err)
					return
				}

				if !c.Conf.Quiet {
					log.Info("Sending message %s", msg)
				}
				if err := streamCli.Send(msg); err != nil {
					errCh <- errors.New(errors.ClientError, "Failed to send message: %v", err)
					return
				}
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
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := streamCli.Recv()
				if err != nil {
					if err == io.EOF {
						// Server ended the stream
						if !c.Conf.Quiet {
							log.Info("Stream closed by server after receiving %d responses", responseCount)
						}
						return
					}
					errCh <- errors.New(errors.ServerError, "Failed to receive response: %v", err)
					return
				}

				if err := c.stream.Send(resp); err != nil {
					errCh <- errors.New(errors.OutputError, "Failed to send response: %v", err)
					return
				}

				responseCount++
			}
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

// handlePingPong handles ping-pong (non-streaming) calls
func (c *StreamingClient) handlePingPong(ctx context.Context) error {
	// Get request data using ioStream
	reqData, err := c.stream.Recv()
	if err != nil && err != io.EOF {
		return errors.New(errors.ClientError, "Failed to read request: %v", err)
	}
	c.Req = reqData

	// For ping-pong, we use the regular GenericCall
	resp, err := c.Client.GenericCall(ctx, c.Conf.Method, c.Req)
	if err != nil {
		// Handle Biz error
		bizErr, isBizErr := kerrors.FromBizStatusError(err)
		if isBizErr {
			if err := c.HandleBizError(bizErr); err != nil {
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
	c.Resp = strResp

	// Send response through ioStream
	if err := c.stream.Send(c.Resp); err != nil {
		return errors.New(errors.OutputError, "Failed to send response: %v", err)
	}

	if !c.Conf.Quiet {
		log.Success()
	}

	return nil
}

// HandleBizError handles business errors
func (c *StreamingClient) HandleBizError(bizErr kerrors.BizStatusErrorIface) error {
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
