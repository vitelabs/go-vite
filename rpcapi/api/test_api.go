package api

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/ledger/generator"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/wallet"
)

type CreateTxWithPrivKeyParmsTest struct {
	SelfAddr    types.Address
	ToAddr      types.Address
	TokenTypeId types.TokenTypeId
	PrivateKey  string
	Amount      string
	Data        []byte
	Difficulty  *big.Int
}

type TestApi struct {
	walletApi *WalletApi
}

func NewTestApi(walletApi *WalletApi) *TestApi {
	return &TestApi{
		walletApi: walletApi,
	}
}

func (t TestApi) CreateTxWithPrivKey(params CreateTxWithPrivKeyParmsTest) error {
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return ErrStrToBigInt
	}
	if err := checkTokenIdValid(t.walletApi.chain, &params.TokenTypeId); err != nil {
		return err
	}
	if !checkTxToAddressAvailable(params.ToAddr) {
		return errors.New("ToAddress is invalid")
	}
	if params.ToAddr == types.AddressDexFund && !dex.VerifyNewOrderPriceForRpc(params.Data) {
		return dex.InvalidOrderPriceErr
	}

	msg := &interfaces.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: params.SelfAddr,
		ToAddress:      &params.ToAddr,
		TokenId:        &params.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Data:           params.Data,
		Difficulty:     params.Difficulty,
	}

	addrState, err := generator.GetAddressStateForGenerator(t.walletApi.chain, &msg.AccountAddress)
	if err != nil || addrState == nil {
		return fmt.Errorf("failed to get addr state for generator, err:%v", err)
	}
	g, e := generator.NewGenerator(t.walletApi.chain, t.walletApi.consensus, msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return e
	}
	account, err := wallet.NewAccountFromHexKey(params.PrivateKey)
	if err != nil {
		return err
	}
	if account.Address() != msg.AccountAddress {
		return errors.New("error private key")
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress, account.Sign)
	if e != nil {
		return e
	}
	if result.Err != nil {
		return result.Err
	}
	if result.VMBlock != nil {
		return t.walletApi.pool.AddDirectAccountBlock(msg.AccountAddress, result.VMBlock)
	} else {
		return errors.New("generator gen an empty block")
	}
}

type CreateReceiveTxParms struct {
	SelfAddr   types.Address
	FromHash   types.Hash
	PrivKeyStr string
	Difficulty *big.Int
}

func (t TestApi) ReceiveOnroadTx(params CreateReceiveTxParms) error {
	chain := t.walletApi.chain
	pool := t.walletApi.pool

	if types.IsContractAddr(params.SelfAddr) {
		return errors.New("AccountTypeContract can't receiveTx without consensus's control")
	}

	msg := &interfaces.IncomingMessage{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: params.SelfAddr,
		FromBlockHash:  &params.FromHash,
		Difficulty:     params.Difficulty,
	}
	account, err := wallet.NewAccountFromHexKey(params.PrivKeyStr)
	if err != nil {
		return err
	}
	if account.Address() != msg.AccountAddress {
		return errors.New("error private key")
	}

	if msg.FromBlockHash == nil {
		return errors.New("params fromblockhash can't be nil")
	}
	fromBlock, err := chain.GetAccountBlockByHash(*msg.FromBlockHash)
	if fromBlock == nil {
		if err != nil {
			return err
		}
		return errors.New("get sendblock by hash failed")
	}
	if fromBlock.ToAddress != msg.AccountAddress {
		return errors.New("can't receive other address's block")
	}

	addrState, err := generator.GetAddressStateForGenerator(t.walletApi.chain, &msg.AccountAddress)
	if err != nil || addrState == nil {
		return fmt.Errorf("failed to get addr state for generator, err:%v", err)
	}
	g, e := generator.NewGenerator(t.walletApi.chain, t.walletApi.consensus, msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress, account.Sign)

	if e != nil {
		return e
	}
	if result.Err != nil {
		return result.Err
	}
	if result.VMBlock != nil {
		return pool.AddDirectAccountBlock(msg.AccountAddress, result.VMBlock)
	} else {
		return errors.New("generator gen an empty block")
	}
}
