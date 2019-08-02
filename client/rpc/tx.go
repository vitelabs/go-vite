//

package rpc

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

// TxApi ...
type TxApi interface {
	SendRawTx(block *api.AccountBlock) error
	SendTxWithPrivateKey(param api.SendTxWithPrivateKeyParam) (*api.AccountBlock, error)
	CalcPoWDifficulty(param api.CalcPoWDifficultyParam) (result *api.CalcPoWDifficultyResult, err error)
}

type txApi struct {
	cc *rpc.Client
}

func NewTxApi(cc *rpc.Client) TxApi {
	return &txApi{cc: cc}
}

func (ti txApi) SendRawTx(block *api.AccountBlock) (err error) {
	err = ti.cc.Call(nil, "tx_sendRawTx", block)
	return
}

func (ti txApi) SendTxWithPrivateKey(param api.SendTxWithPrivateKeyParam) (block *api.AccountBlock, err error) {
	block = &api.AccountBlock{}
	err = ti.cc.Call(block, "tx_sendTxWithPrivateKey", param)
	return
}

func (ti txApi) CalcPoWDifficulty(param api.CalcPoWDifficultyParam) (result *api.CalcPoWDifficultyResult, err error) {
	result = &api.CalcPoWDifficultyResult{}
	err = ti.cc.Call(result, "tx_calcPoWDifficulty", param)
	return
}
