package client

import (
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/rpcapi/api"

	"github.com/vitelabs/go-vite/common/types"
)

type AccountBlock struct {
}

type RequestTxParams struct {
	ToAddr   types.Address
	SelfAddr types.Address
	Amount   *big.Int
	TokenId  types.TokenTypeId
	Data     []byte
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
	SubmitRequestTx(params RequestTxParams, prev *ledger.HashHeight, f SignFunc) (*ledger.HashHeight, error)
	SubmitRequestTxWithPow(params RequestTxParams, f SignFunc) error
	SubmitResponseTx(params ResponseTxParams, f SignFunc) error
	SubmitResponseTxWithPow(params ResponseTxParams, f SignFunc) error
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

func (c *client) SubmitRequestTx(params RequestTxParams, prev *ledger.HashHeight, f SignFunc) (*ledger.HashHeight, error) {
	if prev == nil {
		latest, err := c.rpc.GetLatest(params.SelfAddr)

		if err != nil {
			return nil, err
		}
		prevHeight := uint64(0)
		if latest.Height != "" {
			prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
			if err != nil {
				return nil, err
			}
		}
		prev = &ledger.HashHeight{Height: prevHeight, Hash: latest.Hash}
	}

	amount := params.Amount.String()

	b := api.NormalRequestRawTxParam{
		BlockType:      ledger.BlockTypeSendCall,
		Hash:           types.Hash{},
		PrevHash:       prev.Hash,
		AccountAddress: params.SelfAddr,
		PublicKey:      nil,
		ToAddress:      params.ToAddr,
		TokenId:        params.TokenId,
		Data:           params.Data,
		Nonce:          nil,
		Signature:      nil,
		Height:         strconv.FormatUint(prev.Height+1, 10),
		Amount:         amount,
		Difficulty:     nil,
	}

	accBlock, err := b.LedgerAccountBlock()
	if err != nil {
		return nil, err
	}
	b.Hash = accBlock.ComputeHash()

	signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	if err != nil {
		return nil, err
	}
	b.Signature = signedData
	b.PublicKey = pubkey
	err = c.rpc.SubmitRaw(b)
	if err != nil {
		return nil, err
	}
	return &ledger.HashHeight{Hash: b.Hash, Height: prev.Height + 1}, nil
}

func (c *client) SubmitRequestTxWithPow(params RequestTxParams, f SignFunc) error {
	//latest, err := c.rpc.GetLatest(params.SelfAddr)
	//
	//if err != nil {
	//	return err
	//}
	//if params.SnapshotHash == nil {
	//	referSnapshot, err := c.rpc.GetFittestSnapshot()
	//	if err != nil {
	//		return err
	//	}
	//	params.SnapshotHash = referSnapshot
	//}
	//
	//amount := params.Amount.String()
	//
	//var prevHash types.Hash
	//prevHeight := uint64(0)
	//if latest.Height != "" {
	//	prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
	//	if err != nil {
	//		return err
	//	}
	//	prevHash = latest.Hash
	//}
	//b := RawBlock{
	//	BlockType:      ledger.BlockTypeSendCall,
	//	Hash:           types.Hash{},
	//	PrevHash:       prevHash,
	//	AccountAddress: params.SelfAddr,
	//	PublicKey:      nil,
	//	ToAddress:      params.ToAddr,
	//	FromBlockHash:  types.Hash{},
	//	TokenId:        params.TokenId,
	//	SnapshotHash:   *params.SnapshotHash,
	//	Data:           params.Data,
	//	Nonce:          nil,
	//	Signature:      nil,
	//	FromAddress:    types.Address{},
	//	Height:         strconv.FormatUint(prevHeight+1, 10),
	//	Amount:         &amount,
	//	Fee:            nil,
	//	Difficulty:     nil,
	//	Timestamp:      time.Now().Unix(),
	//}
	//
	//difficulty, err := c.rpc.GetDifficulty(DifficultyQuery{
	//	SelfAddr:       b.AccountAddress,
	//	PrevHash:       b.PrevHash,
	//	SnapshotHash:   b.SnapshotHash,
	//	BlockType:      b.BlockType,
	//	ToAddr:         &b.ToAddress,
	//	Data:           b.Data,
	//	UsePledgeQuota: false,
	//})
	//if err != nil {
	//	return err
	//}
	//difficultyInt, _ := big.NewInt(0).SetString(difficulty, 10)
	//nonce, err := pow.GetPowNonce(difficultyInt, types.DataHash(append(b.AccountAddress.Bytes(), b.PrevHash.Bytes()...)))
	//if err != nil {
	//	return err
	//}
	//b.Nonce = nonce
	//b.Difficulty = &difficulty
	//
	//hashes, err := b.ComputeHash()
	//if err != nil {
	//	return err
	//}
	//b.Hash = *hashes
	//
	//signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	//if err != nil {
	//	return err
	//}
	//b.Signature = signedData
	//b.PublicKey = pubkey
	//
	//return c.rpc.SubmitRaw(b)
	panic("implement")
	return nil
}

func (c *client) SubmitResponseTx(params ResponseTxParams, f SignFunc) error {
	panic("implement")
	//latest, err := c.rpc.GetLatest(params.SelfAddr)
	//
	//if err != nil {
	//	return err
	//}
	//
	//if params.SnapshotHash == nil {
	//	referSnapshot, err := c.rpc.GetFittestSnapshot()
	//	if err != nil {
	//		return err
	//	}
	//	params.SnapshotHash = referSnapshot
	//}
	//reqBlock, err := c.rpc.GetAccBlock(params.RequestHash)
	//if err != nil {
	//	return err
	//}
	//prevHeight := uint64(0)
	//var prevHash types.Hash
	//if latest.Height != "" {
	//	prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
	//	if err != nil {
	//		return err
	//	}
	//	prevHash = latest.Hash
	//}
	//
	//b := RawBlock{
	//	BlockType:      ledger.BlockTypeReceive,
	//	Hash:           types.Hash{},
	//	PrevHash:       prevHash,
	//	AccountAddress: params.SelfAddr,
	//	PublicKey:      nil,
	//	ToAddress:      params.SelfAddr,
	//	FromBlockHash:  reqBlock.Hash,
	//	TokenId:        reqBlock.TokenId,
	//	SnapshotHash:   *params.SnapshotHash,
	//	Data:           nil,
	//	Nonce:          nil,
	//	Signature:      nil,
	//	FromAddress:    reqBlock.AccountAddress,
	//	Height:         strconv.FormatUint(prevHeight+1, 10),
	//	Amount:         &reqBlock.Amount,
	//	Fee:            nil,
	//	Difficulty:     nil,
	//	Timestamp:      time.Now().Unix(),
	//}
	//
	//hashes, err := b.ComputeHash()
	//if err != nil {
	//	return err
	//}
	//b.Hash = *hashes
	//
	//signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	//if err != nil {
	//	return err
	//}
	//b.Signature = signedData
	//b.PublicKey = pubkey
	//
	//return c.rpc.SubmitRaw(b)
	return nil
}

func (c *client) SubmitResponseTxWithPow(params ResponseTxParams, f SignFunc) error {
	panic("implement")
	//latest, err := c.rpc.GetLatest(params.SelfAddr)
	//
	//if err != nil {
	//	return err
	//}
	//
	//if params.SnapshotHash == nil {
	//	referSnapshot, err := c.rpc.GetFittestSnapshot()
	//	if err != nil {
	//		return err
	//	}
	//	params.SnapshotHash = referSnapshot
	//}
	//reqBlock, err := c.rpc.GetAccBlock(params.RequestHash)
	//if err != nil {
	//	return err
	//}
	//prevHeight := uint64(0)
	//var prevHash types.Hash
	//if latest.Height != "" {
	//	prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
	//	if err != nil {
	//		return err
	//	}
	//	prevHash = latest.Hash
	//}
	//
	//b := RawBlock{
	//	BlockType:      ledger.BlockTypeReceive,
	//	Hash:           types.Hash{},
	//	PrevHash:       prevHash,
	//	AccountAddress: params.SelfAddr,
	//	PublicKey:      nil,
	//	ToAddress:      params.SelfAddr,
	//	FromBlockHash:  reqBlock.Hash,
	//	TokenId:        reqBlock.TokenId,
	//	SnapshotHash:   *params.SnapshotHash,
	//	Data:           nil,
	//	Nonce:          nil,
	//	Signature:      nil,
	//	FromAddress:    reqBlock.AccountAddress,
	//	Height:         strconv.FormatUint(prevHeight+1, 10),
	//	Amount:         &reqBlock.Amount,
	//	Fee:            nil,
	//	Difficulty:     nil,
	//	Timestamp:      time.Now().Unix(),
	//}
	//
	//difficulty, err := c.rpc.GetDifficulty(DifficultyQuery{
	//	SelfAddr:       b.AccountAddress,
	//	PrevHash:       b.PrevHash,
	//	SnapshotHash:   b.SnapshotHash,
	//	BlockType:      b.BlockType,
	//	ToAddr:         &b.ToAddress,
	//	Data:           b.Data,
	//	UsePledgeQuota: false,
	//})
	//
	//if err != nil {
	//	return err
	//}
	//
	//difficultyInt, _ := big.NewInt(0).SetString(difficulty, 10)
	//nonce, err := pow.GetPowNonce(difficultyInt, types.DataHash(append(b.AccountAddress.Bytes(), b.PrevHash.Bytes()...)))
	//if err != nil {
	//	return err
	//}
	//b.Nonce = nonce
	//b.Difficulty = &difficulty
	//
	//hashes, err := b.ComputeHash()
	//if err != nil {
	//	return err
	//}
	//b.Hash = *hashes
	//
	//signedData, pubkey, err := f(params.SelfAddr, b.Hash.Bytes())
	//if err != nil {
	//	return err
	//}
	//b.Signature = signedData
	//b.PublicKey = pubkey
	//
	//return c.rpc.SubmitRaw(b)
	return nil
}

func (c *client) QueryOnroad(query OnroadQuery) ([]*AccBlockHeader, error) {
	panic("implement")
	return c.rpc.GetOnroad(query)
}

func (c *client) Balance(query BalanceQuery) (*TokenBalance, error) {
	panic("implement")
	return c.rpc.Balance(query)
}

func (c *client) BalanceAll(query BalanceAllQuery) ([]*TokenBalance, error) {
	panic("implement")
	return c.rpc.BalanceAll(query)
}
