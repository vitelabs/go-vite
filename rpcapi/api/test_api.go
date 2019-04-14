package api

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
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

func (t TestApi) Bazinga(ctx context.Context) error {
	return CheckGetTestTokenIpFrequency(testapi_testtokenlru, ctx)
}

func (t TestApi) GetTestToken(ctx context.Context, ToAddr types.Address) (string, error) {
	if e := CheckGetTestTokenIpFrequency(testapi_testtokenlru, ctx); e != nil {
		return "", e
	}

	privKey, err := ed25519.HexToPrivateKey(testapi_hexPrivKey)
	if err != nil {
		return "", err
	}
	SelfAddr := types.PrikeyToAddress(privKey)
	a := rand.Int() % 1000
	a += 1
	ba := new(big.Int).SetInt64(int64(a))
	ba.Mul(ba, math.BigPow(10, 18))

	amount := ba.String()
	//tid, _ := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	tid, _ := types.HexToTokenTypeId(testapi_tti)

	err = t.CreateTxWithPrivKey(CreateTxWithPrivKeyParmsTest{
		SelfAddr:    SelfAddr,
		ToAddr:      ToAddr,
		TokenTypeId: tid,
		PrivateKey:  testapi_hexPrivKey,
		Amount:      amount,
	})
	if err != nil {
		return "", err
	}

	return amount, nil
}

func (t TestApi) CreateTxWithPrivKey(params CreateTxWithPrivKeyParmsTest) error {
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return ErrStrToBigInt
	}

	msg := &generator.IncomingMessage{
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
		return errors.New(fmt.Sprintf("failed to get addr state for generator, err:%v", err))
	}
	g, e := generator.NewGenerator2(t.walletApi.chain, t.walletApi.consensus, msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		var privkey ed25519.PrivateKey
		privkey, e := ed25519.HexToPrivateKey(params.PrivateKey)
		if e != nil {
			return nil, nil, e
		}
		signData := ed25519.Sign(privkey, data)
		pubkey = privkey.PubByte()
		return signData, pubkey, nil
	})
	if e != nil {
		return e
	}
	if result.Err != nil {
		return result.Err
	}
	if result.VmBlock != nil {
		return t.walletApi.pool.AddDirectAccountBlock(msg.AccountAddress, result.VmBlock)
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

	isContract, err := t.walletApi.chain.IsContractAccount(params.SelfAddr)
	if err != nil {
		return err
	}
	if isContract {
		return errors.New("AccountTypeContract can't receiveTx without consensus's control")
	}

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: params.SelfAddr,
		FromBlockHash:  &params.FromHash,
		Difficulty:     params.Difficulty,
	}
	privKey, err := ed25519.HexToPrivateKey(params.PrivKeyStr)
	if err != nil {
		return err
	}
	pubKey := privKey.PubByte()

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
		return errors.New(fmt.Sprintf("failed to get addr state for generator, err:%v", err))
	}
	g, e := generator.NewGenerator2(t.walletApi.chain, t.walletApi.consensus, msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return ed25519.Sign(privKey, data), pubKey, nil
		})
	if e != nil {
		return e
	}
	if result.Err != nil {
		return result.Err
	}
	if result.VmBlock != nil {
		return pool.AddDirectAccountBlock(msg.AccountAddress, result.VmBlock)
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}
