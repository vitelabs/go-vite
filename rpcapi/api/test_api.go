package api

import (
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"math/rand"
)

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
