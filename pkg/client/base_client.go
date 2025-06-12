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
	"io"
	"os"
	"strings"

	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/remote"
	remote_transmeta "github.com/cloudwego/kitex/pkg/remote/transmeta"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

// BaseClient provides common functionality for all client types
type BaseClient struct {
	Client       genericclient.Client
	Generic      generic.Generic
	Conf         *config.Config
	ClientOpts   []client.Option
	Req          interface{}
	Resp         interface{}
	MetaBackward map[string]string
	stream       *ioStream
}

// Init initializes the base client with common functionality
func (c *BaseClient) Init(conf *config.Config) error {
	c.Conf = conf

	// 1. Initialize ioStream
	if err := c.initIoStream(); err != nil {
		return err
	}

	// 2. Initialize Generic (Thrift/Protobuf)
	if err := c.initGeneric(); err != nil {
		return err
	}

	// 3. Build client options
	if err := c.buildClientOptions(); err != nil {
		return err
	}

	return nil
}

// initIoStream initializes the input/output stream
func (c *BaseClient) initIoStream() error {
	var in io.Reader
	if c.Conf.File != "" {
		file, err := os.Open(c.Conf.File)
		if err != nil {
			return err
		}
		in = file
	} else if c.Conf.Data != "" {
		in = strings.NewReader(c.Conf.Data)
	} else {
		in = os.Stdin
	}
	c.stream = newIoStream(in, os.Stdout)
	return nil
}

// initGeneric initializes the generic provider based on IDL type
func (c *BaseClient) initGeneric() error {
	switch c.Conf.Type {
	case config.Thrift:
		return c.initThriftGeneric()
	case config.Protobuf:
		return c.initPbGeneric()
	default:
		return c.initThriftGeneric()
	}
}

// initThriftGeneric initializes Thrift generic provider
func (c *BaseClient) initThriftGeneric() error {
	p, err := generic.NewThriftFileProvider(c.Conf.IDLPath, c.Conf.IncludePath...)
	if err != nil {
		return err
	}

	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g
	return nil
}

// initPbGeneric initializes Protobuf generic provider
func (c *BaseClient) initPbGeneric() error {
	dOpts := proto.Options{}
	p, err := generic.NewPbFileProviderWithDynamicGo(c.Conf.IDLPath, context.Background(), dOpts, c.Conf.IncludePath...)
	if err != nil {
		return err
	}

	g, err := generic.JSONPbGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g
	return nil
}

// buildClientOptions builds client options for all client types
func (c *BaseClient) buildClientOptions() error {
	var opts []client.Option

	// 1. Basic options
	opts = append(opts, client.WithHostPorts(c.Conf.Endpoint...))

	// 2. Handle metadata
	if c.Conf.Meta != nil || c.Conf.MetaPersistent != nil || c.Conf.MetaBackward {
		opts = append(opts, client.WithMetaHandler(transmeta.ClientHTTP2Handler))
	}

	// 3. Handle transport protocol
	if c.Conf.IsStreaming {
		// For streaming, always use gRPC transport protocol
		opts = append(opts, client.WithTransportProtocol(transport.GRPC))
	} else {
		// For non-streaming calls
		if c.Conf.BizError {
			if c.Conf.Transport != "" && c.Conf.Transport != "TTHeader" {
				return errors.New(errors.ClientError, "It looks like the transport protocol does not support transmitting biz error, use TTHeader please")
			}
			opts = append(opts, client.WithTransportProtocol(transport.TTHeader))
			opts = append(opts, client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
		} else {
			if c.Conf.Transport != "" {
				switch c.Conf.Transport {
				case "TTHeader":
					opts = append(opts, client.WithTransportProtocol(transport.TTHeader))
				case "Framed":
					opts = append(opts, client.WithTransportProtocol(transport.Framed))
				case "TTHeaderFramed":
					opts = append(opts, client.WithTransportProtocol(transport.TTHeaderFramed))
				}
			}
		}

		// 4. Handle IDL service name for TTHeader (only for non-streaming calls)
		if (c.Conf.Transport == "TTHeader" || c.Conf.Transport == "TTHeaderFramed") && c.Conf.IDLServiceName != "" {
			opts = append(opts, client.WithMetaHandler(remote.NewCustomMetaHandler(
				remote.WithWriteMeta(func(ctx context.Context, msg remote.Message) (context.Context, error) {
					strInfo := msg.TransInfo().TransStrInfo()
					strInfo[remote_transmeta.HeaderIDLServiceName] = c.Conf.IDLServiceName
					return ctx, nil
				}),
			)))
		}
	}

	c.ClientOpts = opts
	return nil
}

// GetResponse returns the response
func (c *BaseClient) GetResponse() interface{} {
	return c.Resp
}

// GetMetaBackward returns the metadata received from the server
func (c *BaseClient) GetMetaBackward() map[string]string {
	if c.MetaBackward == nil {
		return make(map[string]string)
	}
	return c.MetaBackward
}

// HandleMetadata handles the metadata output for both streaming and non-streaming calls
func (c *BaseClient) HandleMetadata(metadata map[string]string) error {
	if !c.Conf.MetaBackward || len(metadata) == 0 {
		return nil
	}

	metaBackward, err := log.FormatMap(metadata)
	if err != nil {
		return err
	}

	log.Println("\033[32mReceived metainfo from server: \033[0m")
	log.Println(metaBackward)
	return nil
}
