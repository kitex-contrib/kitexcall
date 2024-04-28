package client

import (
	"context"
	"fmt"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/kitex-contrib/kitexcall/pkg/config"
)

type ThriftGeneric struct {
	// Provider generic.DescriptorProvider
	// Client   genericclient.Client
	// Generic  generic.Generic
}

func NewThriftGeneric() Client {
	return &ThriftGeneric{}
}

func (c *ThriftGeneric) Init(conf config.Config) error {
	p, err := generic.NewThriftFileProvider(conf.IDLPath)
	if err != nil {
		return err
	}

	// Generic calls to construct JSON requests and return types
	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		return err
	}

	opts, err := c.BuildOptions(conf)

	cli, err := genericclient.NewClient(conf.Service, g, opts...)

	// 'ExampleMethod' method name must be included in the idl definition
	resp, err := cli.GenericCall(context.Background(), "ExampleMethod", "{\"Msg\": \"hello\"}")
	if err != nil {
		panic(err)
	}
	// resp is a JSON string
	fmt.Println(resp)

	return nil
}

func (c *ThriftGeneric) Call() {

}

func (c *ThriftGeneric) BuildOptions(conf config.Config) ([]client.Option, error) {
	var opts []client.Option
	opts = append(opts, client.WithHostPorts(conf.Endpoint...))
	return opts, nil
}
