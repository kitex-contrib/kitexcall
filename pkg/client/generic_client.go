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
	"os"

	"github.com/bytedance/gopkg/cloud/metainfo"
	dproto "github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type GenericClientBase struct {
	Client       genericclient.Client
	Generic      generic.Generic
	Conf         *config.Config
	ClientOpts   []client.Option
	CallOptions  []callopt.Option
	Req          interface{}
	Resp         interface{}
	MetaBackward map[string]string
}

func (c *GenericClientBase) Call() error {
	ctx, err := c.BuildCallOptions()
	if err != nil {
		return err
	}

	resp, err := c.Client.GenericCall(ctx, c.Conf.Method, c.Conf.Data, c.CallOptions...)
	if err != nil {
		return err
	}

	if c.Conf.MetaBackward {
		// Receive all meta information from server side
		c.MetaBackward = metainfo.RecvAllBackwardValues(ctx)
	}

	c.Resp = resp
	return nil
}

func (c *GenericClientBase) Output() error {
	var metaBackward string
	var err error
	// Backward metainfo
	if c.Conf.MetaBackward {
		metaBackward, err = log.FormatMap(c.MetaBackward)
		if err != nil {
			return err
		}
	}

	result, err := log.FormatJson(c.Resp.(string))
	if err != nil {
		return err
	}

	log.Success()
	log.Println(result)

	if c.Conf.MetaBackward {
		log.Println("\033[32mReceived metainfo from server: \033[0m")
		log.Println(metaBackward)
	}
	return nil
}

func (c *GenericClientBase) BuildRequest() error {
	if c.Conf.Data != "" {
		c.Req = c.Conf.Data
	} else if c.Conf.File != "" {
		// Read json from file
		data, err := os.ReadFile(c.Conf.File)
		if err != nil {
			return errors.New(errors.ClientError, "failed to read file: %v", err)
		}
		c.Req = string(data)
	}
	return nil
}

func (c *GenericClientBase) BuildClientOptions() error {
	var opts []client.Option
	opts = append(opts, client.WithHostPorts(c.Conf.Endpoint...))

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

	c.ClientOpts = opts
	return nil
}

func (c *GenericClientBase) BuildCallOptions() (context.Context, error) {
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

func (c *GenericClientBase) HandleBizError(bizErr kerrors.BizStatusErrorIface) error {
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

func (c *GenericClientBase) GetResponse() interface{} {
	return c.Resp
}

func (c *GenericClientBase) GetMetaBackward() map[string]string {
	return c.MetaBackward
}

type ThriftGeneric struct {
	GenericClientBase
	Provider generic.DescriptorProvider
}

func NewThriftGeneric() *ThriftGeneric {
	return &ThriftGeneric{}
}

func (c *ThriftGeneric) Init(Conf *config.Config) error {
	// Waiting for server reflection
	p, err := generic.NewThriftFileProvider(Conf.IDLPath)
	if err != nil {
		return err
	}
	c.Provider, c.Conf = p, Conf

	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g

	if err := c.BuildClientOptions(); err != nil {
		return err
	}

	cli, err := genericclient.NewClient(Conf.IDLServiceName, c.Generic, c.ClientOpts...)
	if err != nil {
		return err
	}
	c.Client = cli
	return nil
}

type PbGeneric struct {
	GenericClientBase
	Provider generic.PbDescriptorProviderDynamicGo
}

func NewPbGeneric() *PbGeneric {
	return &PbGeneric{}
}

func (c *PbGeneric) Init(Conf *config.Config) error {
	// Waiting for server reflection
	dOpts := dproto.Options{}
	p, err := generic.NewPbFileProviderWithDynamicGo(Conf.IDLPath, context.Background(), dOpts)
	if err != nil {
		return err
	}
	c.Provider, c.Conf = p, Conf

	g, err := generic.JSONPbGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g

	if err := c.BuildClientOptions(); err != nil {
		return err
	}

	cli, err := genericclient.NewClient(Conf.IDLServiceName, c.Generic, c.ClientOpts...)
	if err != nil {
		return err
	}
	c.Client = cli
	return nil
}
