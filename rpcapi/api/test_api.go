package api

import (
	"context"
	"errors"
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
	_, fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(t.walletApi.chain, &msg.AccountAddress, nil, true)
	if err != nil {
		return err
	}
	g, e := generator.NewGenerator(t.walletApi.chain, fitestSnapshotBlockHash, nil, &params.SelfAddr)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
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
		newerr, _ := TryMakeConcernedError(e)
		return newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		return t.walletApi.pool.AddDirectAccountBlock(params.SelfAddr, result.BlockGenList[0])
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

	code, err := chain.AccountType(&params.SelfAddr)
	if err != nil {
		return err
	}
	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: params.SelfAddr,
		FromBlockHash:  &params.FromHash,
		Difficulty:     params.Difficulty,
	}
	if code == ledger.AccountTypeContract && msg.BlockType == ledger.BlockTypeReceive {
		return errors.New("AccountTypeContract can't receiveTx without consensus's control")
	}
	privKey, err := ed25519.HexToPrivateKey(params.PrivKeyStr)
	if err != nil {
		return err
	}
	pubKey := privKey.PubByte()

	if msg.FromBlockHash == nil {
		return errors.New("params fromblockhash can't be nil")
	}
	fromBlock, err := t.walletApi.chain.GetAccountBlockByHash(msg.FromBlockHash)
	if fromBlock == nil {
		if err != nil {
			return err
		}
		return errors.New("get sendblock by hash failed")
	}

	if fromBlock.ToAddress != params.SelfAddr {
		return errors.New("can't receive other address's block")
	}

	var referredSnapshotHashList []types.Hash
	referredSnapshotHashList = append(referredSnapshotHashList, fromBlock.SnapshotHash)
	_, fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(t.walletApi.chain, &msg.AccountAddress, referredSnapshotHashList, true)
	if err != nil {
		return err
	}
	g, e := generator.NewGenerator(chain, fitestSnapshotBlockHash, nil, &params.SelfAddr)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(privKey, data), pubKey, nil
	})
	if e != nil {
		newerr, _ := TryMakeConcernedError(e)
		return newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		return pool.AddDirectAccountBlock(params.SelfAddr, result.BlockGenList[0])
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}
