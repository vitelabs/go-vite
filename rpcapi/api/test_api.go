package api

import (
	"errors"
	"math/big"
	"math/rand"

	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
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

func (t TestApi) GetTestToken(toAddress types.Address) (string, error) {
	addresses, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	a := rand.Int() % 1000
	a += 1
	ba := new(big.Int).SetInt64(int64(a))
	ba.Mul(ba, math.BigPow(10, 18))

	amount := ba.String()
	tid, _ := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")

	e := t.walletApi.CreateTxWithPassphrase(CreateTransferTxParms{
		SelfAddr:    addresses,
		ToAddr:      toAddress,
		TokenTypeId: tid,
		Passphrase:  "123456",
		Amount:      amount,
	})
	return amount, e
}

func (t TestApi) CreateTxWithPrivKey(params CreateTxWithPrivKeyParmsTest) error {
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return ErrStrToBigInt
	}

	block, e := t.walletApi.chain.GetLatestAccountBlock(&params.SelfAddr)
	if e != nil {
		return e
	}
	preHash := types.Hash{}
	if block != nil {
		preHash = block.Hash
	}

	nonce := pow.GetPowNonce(params.Difficulty, types.DataListHash(params.SelfAddr[:], preHash[:]))

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: params.SelfAddr,
		ToAddress:      &params.ToAddr,
		TokenId:        &params.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Nonce:          nonce[:],
		Data:           params.Data,
	}

	g, e := generator.NewGenerator(t.walletApi.chain, nil, nil, &params.SelfAddr)
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
	}
	if code == ledger.AccountTypeContract && msg.BlockType == ledger.BlockTypeReceive {
		return errors.New("AccountTypeContract can't receiveTx without consensus's control")
	}
	privKey, _ := ed25519.HexToPrivateKey(params.PrivKeyStr)
	pubKey := privKey.PubByte()

	g, e := generator.NewGenerator(chain, nil, nil, &params.SelfAddr)
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
