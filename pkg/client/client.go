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
	"github.com/cloudwego/kitex/pkg/kerrors"

	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
)

type Client interface {
	Init(conf *config.Config) error
	Call() error
	HandleBizError(bizErr kerrors.BizStatusErrorIface) error
	Output() error
	GetResponse() interface{}
	GetMetaBackward() map[string]string
}

func InvokeRPC(conf *config.Config) (Client, error) {
	var c Client

	// Choose the appropriate client type based on configuration
	if conf.IsStreaming {
		// For streaming RPCs, use the streaming client
		c = NewStreamingClient()
	} else {
		// For standard RPCs, choose the appropriate client based on IDL type
		switch conf.Type {
		case config.Thrift:
			c = NewThriftGeneric()
		case config.Protobuf:
			c = NewPbGeneric()
		default:
			c = NewThriftGeneric()
		}
	}

	// Initialize the client
	if err := c.Init(conf); err != nil {
		return nil, errors.New(errors.ClientError, "Client init failed: %v", err)
	}

	// Execute the RPC call for all client types
	if err := c.Call(); err != nil {
		// Handle Biz error
		bizErr, isBizErr := kerrors.FromBizStatusError(err)
		if isBizErr {
			if err := c.HandleBizError(bizErr); err != nil {
				return nil, errors.New(errors.OutputError, "BizError parse error: %v", err)
			}
			return c, nil
		}
		return nil, errors.New(errors.ServerError, "RPC call failed: %v", err)
	}

	// Only handle output for non-streaming clients, as streaming clients handle output directly
	if !conf.IsStreaming {
		if err := c.Output(); err != nil {
			return nil, errors.New(errors.OutputError, "Response parse error: %v", err)
		}
	}

	return c, nil
}
