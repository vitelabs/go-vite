package client

import (
	rpc2 "github.com/vitelabs/go-vite/client/rpc"
	"github.com/vitelabs/go-vite/rpc"
)

type RpcClient interface {
	rpc2.LedgerApi
	rpc2.OnroadApi
	rpc2.TxApi

	GetClient() *rpc.Client
}

func NewRpcClient(rawurl string) (RpcClient, error) {
	c, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}

	r := &rpcClient{
		LedgerApi: rpc2.NewLedgerApi(c),
		OnroadApi: rpc2.NewOnroadApi(c),
		TxApi:     rpc2.NewTxApi(c),
		cc:        c,
	}
	return r, nil
}

type rpcClient struct {
	rpc2.LedgerApi
	rpc2.OnroadApi
	rpc2.TxApi
	cc *rpc.Client
}

func (c rpcClient) GetClient() *rpc.Client {
	return c.cc
}
