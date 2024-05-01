package client

import (
	"context"
	"fmt"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type ThriftGeneric struct {
	Provider generic.DescriptorProvider
	Client   genericclient.Client
	Generic  generic.Generic
	conf     *config.Config
	Resp     interface{}
}

func NewThriftGeneric() Client {
	return &ThriftGeneric{}
}

func (c *ThriftGeneric) Init(conf *config.Config) error {
	// Waiting for server reflection
	p, err := generic.NewThriftFileProvider(conf.IDLPath)
	if err != nil {
		return err
	}
	c.Provider = p
	c.conf = conf

	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		return err
	}
	c.Generic = g

	opts, err := c.BuildClientOptions()
	if err != nil {
		return err
	}

	cli, err := genericclient.NewClient(conf.Service, c.Generic, opts...)
	if err != nil {
		return err
	}
	c.Client = cli
	return nil
}

func (c *ThriftGeneric) Call() error {
	opts, err := c.BuildCallOptions()
	if err != nil {
		return err
	}

	resp, err := c.Client.GenericCall(context.Background(), c.conf.Method, c.conf.Data, opts...)
	if err != nil {
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

func (c *ThriftGeneric) HandleBizError(bizErr kerrors.BizStatusErrorIface) error {
	result, err := log.Format(log.JF, bizErr.BizMessage())
	if err != nil {
		return err
	}
	log.Success(fmt.Sprintf("Biz error: \nStatusCode: %v\nMessage: \n",
		bizErr.BizStatusCode()))
	log.Print(result)
	return nil
}

func (c *ThriftGeneric) BuildClientOptions() ([]client.Option, error) {
	var opts []client.Option
	opts = append(opts, client.WithHostPorts(c.conf.Endpoint...))
	if c.conf.Transport != "" {
		switch c.conf.Transport {
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

func (c *ThriftGeneric) BuildCallOptions() ([]callopt.Option, error) {
	var opts []callopt.Option
	// TODO: add call options
	return opts, nil
}
