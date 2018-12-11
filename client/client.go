package client

import (
	"math/big"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
)

type AccountBlock struct {
}

type RequestTxParams struct {
	ToAddr       types.Address
	SelfAddr     types.Address
	Amount       *big.Int
	TokenId      types.TokenTypeId
	SnapshotHash *types.Hash
	Data         []byte
}

type ResponseTxParams struct {
	SelfAddr     types.Address
	RequestHash  types.Hash
	SnapshotHash *types.Hash
}

type OnroadQuery struct {
	Address types.Address
	Index   int
	Cnt     int
}

type BalanceQuery struct {
	Addr    types.Address
	TokenId types.TokenTypeId
}

type BalanceAllQuery struct {
	Addr types.Address
}

type ledgerClient interface {
	SubmitRequestTx(params RequestTxParams, f SignFunc) error
	SubmitRequestTxWithPow(params RequestTxParams) error
	SubmitResponseTx(params ResponseTxParams, f SignFunc) error
	SubmitResponseTxWithPow(params ResponseTxParams) error
	QueryOnroad(query OnroadQuery) ([]*AccBlockHeader, error)
	Balance(query BalanceQuery) (*TokenBalance, error)
	BalanceAll(query BalanceAllQuery) ([]*TokenBalance, error)
}

type Client interface {
	ledgerClient
}

func NewClient(rpc RpcClient) (Client, error) {
	return &client{rpc}, nil
}

type client struct {
	rpc RpcClient
}

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

func (c *client) SubmitRequestTx(params RequestTxParams, f SignFunc) error {
	latest, err := c.rpc.GetLatest(params.SelfAddr)

	if err != nil {
		return err
	}
	if params.SnapshotHash == nil {
		referSnapshot, err := c.rpc.GetFittestSnapshot()
		if err != nil {
			return err
		}
		params.SnapshotHash = referSnapshot
	}

	amount := params.Amount.String()

	prevHeight, err := strconv.ParseUint(latest.Height, 10, 64)
	if err != nil {
		return err
	}
	b := RawBlock{
		BlockType:      ledger.BlockTypeSendCall,
		Hash:           types.Hash{},
		PrevHash:       latest.Hash,
		AccountAddress: params.SelfAddr,
		PublicKey:      nil,
		ToAddress:      params.ToAddr,
		FromBlockHash:  types.Hash{},
		TokenId:        params.TokenId,
		SnapshotHash:   *params.SnapshotHash,
		Data:           nil,
		Nonce:          nil,
		Signature:      nil,
		FromAddress:    types.Address{},
		Height:         strconv.FormatUint(prevHeight+1, 10),
		Amount:         &amount,
		Fee:            nil,
		Difficulty:     nil,
		Timestamp:      time.Now().Unix(),
	}

	hashes, err := b.ComputeHash()
	if err != nil {
		return err
	}
	b.Hash = *hashes

	signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	if err != nil {
		return err
	}
	b.Signature = signedData
	b.PublicKey = pubkey

	return c.rpc.SubmitRaw(b)
}

func (*client) SubmitRequestTxWithPow(params RequestTxParams) error {
	panic("implement me")
}

func (c *client) SubmitResponseTx(params ResponseTxParams, f SignFunc) error {
	latest, err := c.rpc.GetLatest(params.SelfAddr)

	if err != nil {
		return err
	}

	if params.SnapshotHash == nil {
		referSnapshot, err := c.rpc.GetFittestSnapshot()
		if err != nil {
			return err
		}
		params.SnapshotHash = referSnapshot
	}
	reqBlock, err := c.rpc.GetAccBlock(params.RequestHash)
	if err != nil {
		return err
	}
	prevHeight := uint64(0)
	if latest.Height != "" {
		prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
		if err != nil {
			return err
		}
	}

	b := RawBlock{
		BlockType:      ledger.BlockTypeReceive,
		Hash:           types.Hash{},
		PrevHash:       latest.Hash,
		AccountAddress: params.SelfAddr,
		PublicKey:      nil,
		ToAddress:      params.SelfAddr,
		FromBlockHash:  reqBlock.Hash,
		TokenId:        reqBlock.TokenId,
		SnapshotHash:   *params.SnapshotHash,
		Data:           nil,
		Nonce:          nil,
		Signature:      nil,
		FromAddress:    reqBlock.AccountAddress,
		Height:         strconv.FormatUint(prevHeight+1, 10),
		Amount:         &reqBlock.Amount,
		Fee:            nil,
		Difficulty:     nil,
		Timestamp:      time.Now().Unix(),
	}

	hashes, err := b.ComputeHash()
	if err != nil {
		return err
	}
	b.Hash = *hashes

	signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	if err != nil {
		return err
	}
	b.Signature = signedData
	b.PublicKey = pubkey

	return c.rpc.SubmitRaw(b)

}

func (*client) SubmitResponseTxWithPow(params ResponseTxParams) error {
	panic("implement me")
}

func (c *client) QueryOnroad(query OnroadQuery) ([]*AccBlockHeader, error) {
	return c.rpc.GetOnroad(query)
}

func (c *client) Balance(query BalanceQuery) (*TokenBalance, error) {
	return c.rpc.Balance(query)
}

func (c *client) BalanceAll(query BalanceAllQuery) ([]*TokenBalance, error) {
	return c.rpc.BalanceAll(query)
}
