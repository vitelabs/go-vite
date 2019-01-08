package mobile

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/rpc"
)

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
	return vc.rawMsgCall("ledger_getBlocksByAccAddr", addr.address, index, count)
}

func (vc *Client) GetBlocksByHash(addr *Address, startHash string, count int) (string, error) {
	if startHash == "" {
		return vc.rawMsgCall("ledger_getBlocksByHash", addr.address, nil, count)
	}
	return vc.rawMsgCall("ledger_getBlocksByHash", addr.address, startHash, count)

}

func (vc *Client) GetAccountByAccAddr(addr *Address) (string, error) {
	return vc.rawMsgCall("ledger_getAccountByAccAddr", addr.address)
}

func (vc *Client) GetOnroadAccountByAccAddr(addr *Address) (string, error) {
	return vc.rawMsgCall("onroad_getAccountOnroadInfo", addr.address)
}

func (vc *Client) GetOnroadBlocksByAddress(addr *Address, index int, count int) (string, error) {
	return vc.rawMsgCall("onroad_getOnroadBlocksByAddress", addr.address, index, count)
}

func (vc *Client) GetLatestBlock(addr *Address) (string, error) {
	return vc.rawMsgCall("ledger_getLatestBlock", addr.address)
}

func (vc *Client) GetPledgeData(addr *Address) (string, error) {
	return vc.stringCall("pledge_getPledgeData", addr.address)
}

func (vc *Client) GetPledgeQuota(addr *Address) (string, error) {
	return vc.rawMsgCall("pledge_getPledgeQuota", addr.address)
}

func (vc *Client) GetPledgeList(addr *Address, index int, count int) (string, error) {
	return vc.rawMsgCall("pledge_getPledgeList", addr.address, index, count)
}

func (vc *Client) GetFittestSnapshotHash(accAddr *Address, sendBlockHash string) (string, error) {
	if sendBlockHash == "" {
		return vc.stringCall("ledger_getFittestSnapshotHash", accAddr.address)
	}
	return vc.stringCall("ledger_getFittestSnapshotHash", accAddr.address, sendBlockHash)
}

func (vc *Client) GetPowNonce(difficulty string, data string) (string, error) {
	return vc.stringCall("pow_getPowNonce", difficulty, data)
}

func (vc *Client) GetSnapshotChainHeight() (string, error) {
	return vc.stringCall("ledger_getSnapshotChainHeight")
}

func (vc *Client) GetCandidateList(gid string) (string, error) {
	return vc.rawMsgCall("register_getCandidateList", gid)
}

func (vc *Client) GetVoteData(gid string, name string) (string, error) {
	return vc.stringCall("vote_getVoteData", gid, name)
}

func (vc *Client) GetVoteInfo(gid string, addr *Address) (string, error) {
	return vc.rawMsgCall("vote_getVoteInfo", gid, addr.address)
}

func (vc *Client) GetCancelVoteData(gid string) (string, error) {
	return vc.stringCall("vote_getCancelVoteData", gid)
}

func (vc *Client) SendRawTx(accBlock string) error {
	var js json.RawMessage = []byte(accBlock)
	err := vc.c.Call(nil, "tx_sendRawTx", js)
	return makeJsonError(err)
}

func (vc *Client) rawMsgCall(method string, args ...interface{}) (string, error) {
	info := json.RawMessage{}
	err := vc.c.Call(&info, method, args...)
	if err != nil {
		return "", makeJsonError(err)
	}
	return string(info), nil
}

func (vc *Client) stringCall(method string, args ...interface{}) (string, error) {
	info := ""
	err := vc.c.Call(&info, method, args...)
	if err != nil {
		return "", makeJsonError(err)
	}
	return info, nil
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
