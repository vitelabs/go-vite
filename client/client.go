package client

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
)

var errorEmptyHash = errors.New("empty hash")
var errorNilWallet = errors.New("nil wallet")
var errorNilKey = errors.New("nil key")
var errorNilBlock = errors.New("nil block")

type RequestTxParams struct {
	ToAddr   types.Address
	SelfAddr types.Address
	Amount   *big.Int
	TokenId  types.TokenTypeId
	Data     []byte
}

type RequestCreateContractParams struct {
	SelfAddr   types.Address
	fee        *big.Int
	arguments  []interface{}
	abiStr     string
	metaParams api.CreateContractDataParam
}

type ResponseTxParams struct {
	SelfAddr    types.Address
	RequestHash types.Hash
}

type Client interface {
	DexClient
	BuildNormalRequestBlock(params RequestTxParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error)
	BuildRequestCreateContractBlock(params RequestCreateContractParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error)
	BuildResponseBlock(params ResponseTxParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error)
	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, *big.Int, error)
	GetBalanceAll(addr types.Address) (*api.RpcAccountInfo, *api.RpcAccountInfo, error)
	SignData(wallet *entropystore.Manager, block *api.AccountBlock) error
	SignDataWithPriKey(key *derivation.Key, block *api.AccountBlock) error
}

func NewClient(rpc RpcClient) (Client, error) {
	return &client{rpc}, nil
}

type client struct {
	rpc RpcClient
}

type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

func (c *client) BuildNormalRequestBlock(params RequestTxParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error) {
	if prev == nil {
		prev, err = c.getPrev(params.SelfAddr)
		if err != nil {
			return
		}
	}

	amount := params.Amount.String()
	block = &api.AccountBlock{
		BlockType:          ledger.BlockTypeSendCall,
		Hash:               types.Hash{},
		PrevHash:           prev.Hash,
		AccountAddress:     params.SelfAddr,
		PublicKey:          nil,
		ToAddress:          params.ToAddr,
		TokenId:            params.TokenId,
		Data:               params.Data,
		Nonce:              nil,
		Signature:          nil,
		Height:             strconv.FormatUint(prev.Height+1, 10),
		Amount:             &amount,
		Difficulty:         nil,
		TokenInfo:          nil,
		ConfirmedTimes:     nil,
		ConfirmedHash:      nil,
		ReceiveBlockHeight: nil,
		ReceiveBlockHash:   nil,
		Timestamp:          0,
	}

	accBlock, err := block.RpcToLedgerBlock()
	if err != nil {
		return nil, err
	}
	block.Hash = accBlock.ComputeHash()
	return
}

func (c *client) BuildRequestCreateContractBlock(params RequestCreateContractParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error) {
	if prev == nil {
		prev, err = c.getPrev(params.SelfAddr)
		if err != nil {
			return
		}
	}
	contractAddr := util.NewContractAddress(params.SelfAddr, prev.Height+1, prev.Hash)
	abiContract, err := abi.JSONToABIContract(strings.NewReader(params.abiStr))
	if err != nil {
		return nil, err
	}
	constructorParams, err := abiContract.PackMethod("", params.arguments...)
	if err != nil {
		return nil, err
	}
	params.metaParams.Params = constructorParams
	data, err := c.rpc.GetCreateContractData(params.metaParams)
	if err != nil {
		return nil, err
	}

	if params.fee == nil {
		one := big.NewInt(1e18)
		params.fee = one.Mul(one, big.NewInt(10))
	}
	fee := params.fee.String()
	block = &api.AccountBlock{
		BlockType:          ledger.BlockTypeSendCreate,
		Hash:               types.Hash{},
		PrevHash:           prev.Hash,
		AccountAddress:     params.SelfAddr,
		PublicKey:          nil,
		ToAddress:          contractAddr,
		Data:               data,
		Nonce:              nil,
		Signature:          nil,
		Fee:                &fee,
		Height:             strconv.FormatUint(prev.Height+1, 10),
		Difficulty:         nil,
		TokenInfo:          nil,
		ConfirmedTimes:     nil,
		ConfirmedHash:      nil,
		ReceiveBlockHeight: nil,
		ReceiveBlockHash:   nil,
		Timestamp:          0,
	}

	accBlock, err := block.RpcToLedgerBlock()
	if err != nil {
		return nil, err
	}
	block.Hash = accBlock.ComputeHash()
	return
}

func (c *client) BuildResponseBlock(params ResponseTxParams, prev *ledger.HashHeight) (block *api.AccountBlock, err error) {
	if prev == nil {
		prev, err = c.getPrev(params.SelfAddr)
		if err != nil {
			return
		}
	}

	block = &api.AccountBlock{
		BlockType:          ledger.BlockTypeReceive,
		Hash:               types.Hash{},
		PrevHash:           prev.Hash,
		AccountAddress:     params.SelfAddr,
		PublicKey:          nil,
		FromBlockHash:      params.RequestHash,
		Data:               nil,
		Nonce:              nil,
		Signature:          nil,
		Height:             strconv.FormatUint(prev.Height+1, 10),
		Fee:                nil,
		Difficulty:         nil,
		TokenInfo:          nil,
		ConfirmedTimes:     nil,
		ConfirmedHash:      nil,
		ReceiveBlockHeight: nil,
		ReceiveBlockHash:   nil,
	}

	accountBlock, err := block.RpcToLedgerBlock()
	if err != nil {
		return nil, err
	}
	hash := accountBlock.ComputeHash()

	block.Hash = hash
	return
}

func (c *client) GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, *big.Int, error) {
	allBalance, allOnroad, err := c.GetBalanceAll(addr)
	if err != nil {
		return nil, nil, err
	}
	balance := big.NewInt(0)
	onroad := big.NewInt(0)

	if allBalance.TokenBalanceInfoMap != nil {
		if info, ok := allBalance.TokenBalanceInfoMap[tokenId]; ok {
			balance.SetString(info.TotalAmount, 10)
		}
	}
	if allOnroad.TokenBalanceInfoMap != nil {
		if info, ok := allOnroad.TokenBalanceInfoMap[tokenId]; ok {
			onroad.SetString(info.TotalAmount, 10)
		}
	}
	return balance, onroad, err
}

func (c *client) GetBalanceAll(addr types.Address) (*api.RpcAccountInfo, *api.RpcAccountInfo, error) {
	allBalance, err := c.rpc.GetAccountByAccAddr(addr)
	if err != nil {
		return nil, nil, err
	}

	allOnroad, err := c.rpc.GetOnroadInfoByAddress(addr)
	if err != nil {
		return nil, nil, err
	}
	return allBalance, allOnroad, nil
}

func (c *client) getPrev(addr types.Address) (*ledger.HashHeight, error) {
	latest, err := c.rpc.GetLatestBlock(addr)

	if err != nil {
		return nil, err
	}
	prevHeight := uint64(0)
	prevHash := types.Hash{}
	if latest != nil && latest.Height != "" {
		prevHeight, err = strconv.ParseUint(latest.Height, 10, 64)
		if err != nil {
			return nil, err
		}
		prevHash = latest.Hash
	}
	return &ledger.HashHeight{Height: prevHeight, Hash: prevHash}, nil
}

func (c *client) SignData(wallet *entropystore.Manager, block *api.AccountBlock) error {
	if wallet == nil {
		return errorNilWallet
	}
	if block == nil {
		return errorNilBlock
	}
	if (block.Hash == types.Hash{}) {
		return errorEmptyHash
	}

	addr := block.AccountAddress
	signedData, pubkey, err := wallet.SignData(addr, block.Hash.Bytes())
	if err != nil {
		return err
	}
	block.Signature = signedData
	block.PublicKey = pubkey
	return nil
}

func (c *client) SignDataWithPriKey(key *derivation.Key, block *api.AccountBlock) error {
	if key == nil {
		return errorNilKey
	}
	if block == nil {
		return errorNilBlock
	}
	if (block.Hash == types.Hash{}) {
		return errorEmptyHash
	}

	signData, pub, err := key.SignData(block.Hash.Bytes())
	if err != nil {
		return err
	}

	block.Signature = signData
	block.PublicKey = pub
	return nil
}
