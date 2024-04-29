package client

import (
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
)

type Client interface {
	Init(conf config.Config) error
	Call() error
	Output() error
}

func InvokeRPC(c Client, conf config.Config) error {

	if err := c.Init(conf); err != nil {
		return errors.New(errors.ClientError, "client init failed: %v", err)
	}

	if err := c.Call(); err != nil {
		return errors.New(errors.ServerError, "rpc call failed: %v", err)
	}

	if err := c.Output(); err != nil {
		return errors.New(errors.OutputError, "response parse error")
	}

	return nil
}

func handleBizError(err error) error {
	bizErr, isBizErr := kerrors.FromBizStatusError(err)
	if isBizErr {
		return errors.New(errors.BizError, "Biz error: StatusCode: %v, Message: %v",
			bizErr.BizStatusCode(), bizErr.BizMessage())
	}

	return nil
}
