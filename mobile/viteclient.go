package mobile

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

type AccountInfo struct {
	api.RpcAccountInfo
}

type Client struct {
	c *rpc.Client
}

func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

func (vc *Client) Close() {
	vc.c.Close()
}

func (vc *Client) GetBlocksByAccAddr(addr *Address, index int, count int) (string, error) {
	return vc.normalCall("ledger_getBlocksByAccAddr", addr.address, index, count)
}

func (vc *Client) GetAccountByAccAddr(addr *Address) (string, error) {
	return vc.normalCall("ledger_getAccountByAccAddr", addr.address)
}

func (vc *Client) GetOnroadAccountByAccAddr(addr *Address) (string, error) {
	return vc.normalCall("onroad_getAccountOnroadInfo", addr.address)
}

func (vc *Client) GetOnroadBlocksByAddress(addr *Address, index int, count int) (string, error) {
	return vc.normalCall("onroad_getOnroadBlocksByAddress", addr.address, index, count)
}

func (vc *Client) GetLatestBlock(addr *Address) (string, error) {
	return vc.normalCall("ledger_getLatestBlock", addr.address)
}

func (vc *Client) GetFittestSnapshotHash() (string, error) {
	return vc.normalCall("ledger_getFittestSnapshotHash")
}

func (vc *Client) GetPowNonce(difficulty string, data string) (string, error) {
	return vc.normalCall("pow_getPowNonce", difficulty, data)
}

func (vc *Client) GetSnapshotChainHeight() (string, error) {
	return vc.normalCall("ledger_getSnapshotChainHeight")
}

func (vc *Client) SendRawTx(accBlock string) (string, error) {
	var js json.RawMessage = []byte(accBlock)
	return vc.normalCall("tx_sendRawTx", js)
}

func (vc *Client) normalCall(method string, args ...interface{}) (string, error) {
	info := json.RawMessage{}
	err := vc.c.Call(&info, method, args)
	if err != nil {
		return "", makeJsonError(err)
	}
	bytes, e := info.MarshalJSON()
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}

func makeJsonError(err error) error {
	if err == nil {
		return nil
	}
	if jr, ok := err.(*rpc.JsonError); ok {
		bytes, _ := json.Marshal(jr)
		return errors.New(string(bytes))
	}
	return err
}
