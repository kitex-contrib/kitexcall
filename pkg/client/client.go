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

// Client interface defines the methods that all client types must implement
type Client interface {
	Init(conf *config.Config) error
	Call() error
	HandleBizError(bizErr kerrors.BizStatusErrorIface) error
	HandleMetadata(metadata map[string]string) error
	GetMetaBackward() map[string]string
}

// InvokeRPC creates and initializes the appropriate client based on configuration
func InvokeRPC(conf *config.Config) (Client, error) {
	var c Client

	// Choose the appropriate client type based on configuration
	if conf.IsStreaming {
		// For streaming RPCs, use the streaming client
		c = NewStreamingClient()
	} else {
		// For standard RPCs, use the generic client
		c = NewGenericClient()
	}

	// Initialize the client
	if err := c.Init(conf); err != nil {
		return nil, errors.New(errors.ClientError, "Client init failed: %v", err)
	}

	// Execute the RPC call
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

	return c, nil
}
