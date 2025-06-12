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
	"context"
	"io"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

// GenericClient implements the Client interface for generic calls
type GenericClient struct {
	BaseClient
	CallOptions []callopt.Option
}

// NewGenericClient creates a new GenericClient
func NewGenericClient() *GenericClient {
	return &GenericClient{}
}

// Init initializes the GenericClient
func (c *GenericClient) Init(conf *config.Config) error {
	// Initialize base client
	if err := c.BaseClient.Init(conf); err != nil {
		return err
	}

	// Create generic client
	cli, err := genericclient.NewClient(conf.IDLServiceName, c.Generic, c.ClientOpts...)
	if err != nil {
		return err
	}
	c.Client = cli

	return nil
}

// Call executes the generic RPC call
func (c *GenericClient) Call() error {
	ctx, err := c.buildCallOptions()
	if err != nil {
		return err
	}

	// Get request data from ioStream
	reqData, err := c.stream.Recv()
	if err != nil && err != io.EOF {
		return errors.New(errors.ClientError, "Failed to read request: %v", err)
	}
	c.Req = reqData

	resp, err := c.Client.GenericCall(ctx, c.Conf.Method, c.Req, c.CallOptions...)
	if err != nil {
		return err
	}

	if c.Conf.MetaBackward {
		// Receive all meta information from server side
		c.MetaBackward = metainfo.RecvAllBackwardValues(ctx)
	}

	c.Resp = resp

	// Send response through ioStream
	if err := c.stream.Send(c.Resp); err != nil {
		return errors.New(errors.OutputError, "Failed to send response: %v", err)
	}

	if !c.Conf.Quiet {
		log.Success()
	}

	// Handle metadata if present
	if err := c.BaseClient.HandleMetadata(c.MetaBackward); err != nil {
		return err
	}

	return nil
}

// HandleBizError handles business errors
func (c *GenericClient) HandleBizError(bizErr kerrors.BizStatusErrorIface) error {
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

// buildCallOptions builds the context with call options
func (c *GenericClient) buildCallOptions() (context.Context, error) {
	ctx := context.Background()
	var opts []callopt.Option

	// Add metainfo to context
	if c.Conf.Transport != "TTHeader" &&
		(len(c.Conf.Meta) != 0 || len(c.Conf.MetaPersistent) != 0 || c.Conf.MetaBackward) {
		return nil, errors.New(errors.ClientError, "It looks like the protocol does not support transmitting meta information")
	}

	if c.Conf.Transport == "TTHeader" {
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
	}
	c.CallOptions = opts
	return ctx, nil
}
