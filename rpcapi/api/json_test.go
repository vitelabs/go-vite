package api

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
)

type DummyBlock struct {
	AccountAddress *types.Address `json:",omitempty"` // Self account
	ConfirmedTimes *string        `json:",omitempty"` // bigint block`s confirmed times
	T              *big.Int       `json:",omitempty"`
}

func TestJson(t *testing.T) {

	m := make(map[types.TokenTypeId]*RpcTokenBalanceInfo)
	id := types.CreateTokenTypeId([]byte{1, 3, 4})
	totalSupply := "10000"
	pledgeAmount := "10000"
	number := "10000"
	addresses, _, _ := types.CreateAddress()

	m[id] = &RpcTokenBalanceInfo{
		TokenInfo: &RpcTokenInfo{
			TokenName:      "as",
			TokenSymbol:    "aa",
			TotalSupply:    &totalSupply,
			Decimals:       19,
			Owner:          addresses,
			PledgeAmount:   &pledgeAmount,
			WithdrawHeight: "12",
		},
		TotalAmount: "132",
		Number:      &number,
	}
	r := RpcAccountInfo{
		AccountAddress:      addresses,
		TotalNumber:         "433",
		TokenBalanceInfoMap: m,
	}
	bytes, _ := json.Marshal(r)
	fmt.Println(string(bytes))

}
