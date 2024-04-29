package client

import (
	"fmt"

	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type Client interface {
	Init(conf *config.Config) error
	Call() error
	Output() error
}

func InvokeRPC(c Client, conf *config.Config) error {

	if err := c.Init(conf); err != nil {
		return errors.New(errors.ClientError, "client init failed: %v", err)
	}

	if err := c.Call(); err != nil {
		// Handle Biz error
		bizErr, isBizErr := kerrors.FromBizStatusError(err)
		if isBizErr {
			log.Success(fmt.Sprintf("Biz error: StatusCode: %v, Message: %v",
				bizErr.BizStatusCode(), bizErr.BizMessage()))
			return nil
		}
		return errors.New(errors.ServerError, "RPC call failed: %v", err)
	}

	if err := c.Output(); err != nil {
		return errors.New(errors.OutputError, "Response parse error")
	}

	return nil
}
