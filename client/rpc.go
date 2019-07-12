package client

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

type BlockHeader struct {
	Hash     types.Hash `json:"hash"`
	PrevHash types.Hash `json:"prevHash"`

	Height string `json:"height"`

	Timestamp int64 `json:"timestamp"`
}

type Block struct {
	Hash     types.Hash `json:"hash"`
	PrevHash types.Hash `json:"prevHash"`
	Height   uint64     `json:"height"`
}

type AccBlockHeader struct {
	Hash           types.Hash    `json:"hash"`
	PrevHash       types.Hash    `json:"prevHash"`
	AccountAddress types.Address `json:"accountAddress"`
	Height         string        `json:"height"`
	Timestamp      int64         `json:"timestamp"`

	Amount string `json:"amount"`
	TInfo  `json:"tokenInfo"`
}

type TInfo struct {
	TokenId     types.TokenTypeId `json:"tokenId"`
	TokenName   string            `json:"tokenName"`
	TokenSymbol string            `json:"tokenSymbol"`
}

type TokenBalance struct {
	TotalAmount string `json:"totalAmount"`
	TInfo       `json:"tokenInfo"`
}

type RpcClient interface {
	SubmitRaw(block api.NormalRequestRawTxParam) error
	GetLatest(address types.Address) (*BlockHeader, error)
	GetSnapshotLatest() (*BlockHeader, error)
	GetFittestSnapshot() (*types.Hash, error)
	GetSnapshotByHash(hash types.Hash) (*BlockHeader, error)
	GetSnapshotByHeight(height uint64) (*BlockHeader, error)
	GetAccBlock(hash types.Hash) (*AccBlockHeader, error)
	GetOnroad(query OnroadQuery) ([]*AccBlockHeader, error)
	Balance(query BalanceQuery) (*TokenBalance, error)
	BalanceAll(query BalanceAllQuery) ([]*TokenBalance, error)
	GetDifficulty(query DifficultyQuery) (diffculty string, err error)

	GetRewardByIndex(index uint64) (reward *api.RewardInfo, err error)
	GetVoteDetailsByIndex(index uint64) (details []*consensus.VoteDetails, err error)

	GetClient() *rpc.Client
}

func NewRpcClient(rawurl string) (RpcClient, error) {
	c, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}

	r := &rpcClient{cc: c}
	return r, nil
}

type rpcClient struct {
	cc *rpc.Client
}

func (c rpcClient) GetClient() *rpc.Client {
	return c.cc
}

func (c *rpcClient) GetSnapshotLatest() (*BlockHeader, error) {
	header := BlockHeader{}
	err := c.cc.Call(&header, "ledger_getSnapshotChain", nil)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) GetSnapshotByHash(hash types.Hash) (*BlockHeader, error) {
	header := BlockHeader{}
	err := c.cc.Call(&header, "ledger_getSnapshotByHash", hash)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) GetSnapshotByHeight(height uint64) (*BlockHeader, error) {
	header := BlockHeader{}
	err := c.cc.Call(&header, "ledger_getSnapshotByHeight", height)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) BalanceAll(query BalanceAllQuery) ([]*TokenBalance, error) {
	var bs []*TokenBalance
	err := c.cc.Call(&bs, "ledger_getBalanceByAccAddr", query.Addr)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (c *rpcClient) Balance(query BalanceQuery) (*TokenBalance, error) {
	b := TokenBalance{}
	err := c.cc.Call(&b, "ledger_getBalanceByAccAddrToken", query.Addr, query.TokenId)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (c *rpcClient) GetOnroad(query OnroadQuery) ([]*AccBlockHeader, error) {
	var blocks []*AccBlockHeader
	err := c.cc.Call(&blocks, "onroad_getOnroadBlocksByAddress", query.Address, query.Index, query.Cnt)
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

func (c *rpcClient) GetAccBlock(hash types.Hash) (*AccBlockHeader, error) {
	header := AccBlockHeader{}
	err := c.cc.Call(&header, "ledger_getBlockByHash", hash)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) GetFittestSnapshot() (*types.Hash, error) {
	hash := types.Hash{}
	err := c.cc.Call(&hash, "ledger_getFittestSnapshotHash", nil)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}

func (c *rpcClient) GetLatest(address types.Address) (*BlockHeader, error) {
	header := BlockHeader{}
	err := c.cc.Call(&header, "ledger_getLatestBlock", address)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) SubmitRaw(block api.NormalRequestRawTxParam) error {
	err := c.cc.Call(nil, "tx_sendRawTx", &block)
	//err := c.cc.Call(nil, "tx_sendNormalRequestRawTx", &block)
	return err
}

type DifficultyQuery struct {
	SelfAddr     types.Address `json:"selfAddr"`
	PrevHash     types.Hash    `json:"prevHash"`
	SnapshotHash types.Hash    `json:"snapshotHash"`

	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddr"`
	Data      []byte         `json:"data"`

	UsePledgeQuota bool `json:"usePledgeQuota"`
}

func (c *rpcClient) GetDifficulty(query DifficultyQuery) (diffculty string, err error) {
	err = c.cc.Call(&diffculty, "tx_calcPoWDifficulty", &query)
	return
}
