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
}

func InvokeRPC(c Client, conf *config.Config) error {

	if err := c.Init(conf); err != nil {
		return errors.New(errors.ClientError, "client init failed: %v", err)
	}

	// var isBizErr bool
	// var bizErr kerrors.BizStatusErrorIface
	if err := c.Call(); err != nil {
		// Handle Biz error
		bizErr, isBizErr := kerrors.FromBizStatusError(err)
		if isBizErr {
			if err := c.HandleBizError(bizErr); err != nil {
				errors.New(errors.OutputError, "Response parse error")
			}
			return nil
		}
		return errors.New(errors.ServerError, "RPC call failed: %v", err)
	}

	if err := c.Output(); err != nil {
		return errors.New(errors.OutputError, "Response parse error")
	}

	return nil
}
