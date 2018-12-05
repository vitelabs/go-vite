package client

import (
	"math/big"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet/entropystore"
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
}

type OnroadQuery struct {
}

type TokenQuery struct {
}

type BalanceQuery struct {
}

type BalanceListQuery struct {
}

type ledgerClient interface {
	SubmitRequestTx(params RequestTxParams) error
	SubmitRequestTxWithPow(params RequestTxParams) error
	SubmitResponseTx(params ResponseTxParams) error
	SubmitResponseTxWithPow(params ResponseTxParams) error
	QueryOnroad(query OnroadQuery)
	TokenList(query TokenQuery)
	Balance(query BalanceQuery)
	BalanceList(query BalanceListQuery)
}

type Client interface {
	ledgerClient
}

func NewClient(rpc RpcClient, em *entropystore.Manager) (Client, error) {
	unlocked := em.IsUnlocked()
	if !unlocked {
		return nil, errors.New("entropy must be unlock.")
	}
	return &client{rpc, em}, nil
}

type client struct {
	rpc RpcClient
	em  *entropystore.Manager
}

func (c *client) SubmitRequestTx(params RequestTxParams) error {
	latest, err := c.rpc.GetLatest(params.SelfAddr)

	if err != nil {
		return err
	}
	referSnapshot, err := c.rpc.getFittestSnapshot()
	if err != nil {
		return err
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
		SnapshotHash:   *referSnapshot,
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

	signedData, pubkey, err := c.em.SignData(params.SelfAddr, b.Hash.Bytes(), nil, nil)
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

func (*client) SubmitResponseTx(params ResponseTxParams) error {
	panic("implement me")
}

func (*client) SubmitResponseTxWithPow(params ResponseTxParams) error {
	panic("implement me")
}

func (*client) QueryOnroad(query OnroadQuery) {
	panic("implement me")
}

func (*client) TokenList(query TokenQuery) {
	panic("implement me")
}

func (*client) Balance(query BalanceQuery) {
	panic("implement me")
}

func (*client) BalanceList(query BalanceListQuery) {
	panic("implement me")
}
