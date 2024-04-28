package client

import (
	"context"
	"fmt"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
)

type Client interface {
	Init(conf config.Config) error
	Call()
}

func InvokeRPC(c Client, conf config.Config) error {

	err := c.Init(conf)
	if err != nil {
		return errors.New(errors.ClientError, "client init failed: %v", err)
	}

	// c.Call()

	// Local file idl parsing
	// YOUR_IDL_PATH thrift file path: example ./idl/example.thrift
	// includeDirs: Specify the include path. By default, the relative path of the current file is used to find include.
	p, err := generic.NewThriftFileProvider("./example_service.thrift")
	if err != nil {
		panic(err)
	}
	// Generic calls to construct JSON requests and return types
	g, err := generic.JSONThriftGeneric(p)
	if err != nil {
		panic(err)
	}
	cli, err := genericclient.NewClient("destServiceName", g, client.WithHostPorts("0.0.0.0:9999"))
	if err != nil {
		panic(err)
	}
	// 'ExampleMethod' method name must be included in the idl definition
	resp, err := cli.GenericCall(context.Background(), "ExampleMethod", "{\"Msg\": \"hello\"}")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
