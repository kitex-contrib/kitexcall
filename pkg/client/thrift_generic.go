package client

import (
	"context"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type ThriftGeneric struct {
	Provider generic.DescriptorProvider
	Client   genericclient.Client
	Generic  generic.Generic
	conf     config.Config
	Resp     interface{}
}

func NewThriftGeneric() Client {
	return &ThriftGeneric{}
}

func (c *ThriftGeneric) Init(conf config.Config) error {
	// Waiting for server reflection
	p, err := generic.NewThriftFileProvider(conf.IDLPath)
	if err != nil {
		return err
	}
	c.Provider = p

	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g

	opts, err := c.BuildOptions(conf)
	if err != nil {
		return err
	}
	c.conf = conf

	cli, err := genericclient.NewClient(conf.Service, g, opts...)
	if err != nil {
		return err
	}
	c.Client = cli
	return nil
}

func (c *ThriftGeneric) Call() error {
	// 'ExampleMethod' method name must be included in the idl definition
	// resp, err := c.Client.GenericCall(context.Background(), "ExampleMethod", "{\"Msg\": \"hello\"}")
	resp, err := c.Client.GenericCall(context.Background(), c.conf.Method, c.conf.Data)
	if err != nil {
		if handleBizError(err) != nil {
			return err
		}
		return err
	}

	c.Resp = resp
	return nil
}

func (c *ThriftGeneric) Output() error {
	result, err := log.Format(log.JF, c.Resp.(string))
	if err != nil {
		return err
	}

	log.Success(result)
	return nil
}

func (c *ThriftGeneric) BuildOptions(conf config.Config) ([]client.Option, error) {
	var opts []client.Option
	opts = append(opts, client.WithHostPorts(conf.Endpoint...))
	if conf.Transport != "" {
		switch conf.Transport {
		case "TTHeader":
			opts = append(opts, client.WithTransportProtocol(transport.TTHeader))
		case "Framed":
			opts = append(opts, client.WithTransportProtocol(transport.Framed))
		case "TTHeaderFramed":
			opts = append(opts, client.WithTransportProtocol(transport.TTHeaderFramed))
		}
	}

	return opts, nil
}
