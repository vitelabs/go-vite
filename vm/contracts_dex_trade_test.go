package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type OrderStorage struct {
	Id                   int32
	Address              types.Address
	MarketId             int32
	Side                 bool
	Type                 int32
	Price                string
	TakerFeeRate         int32
	MakerFeeRate         int32
	TakerOperatorFeeRate int32
	MakerOperatorFeeRate int32
	Quantity             *big.Int
	Amount               *big.Int
	Status               int32
	ExecutedQuantity     *big.Int
	ExecutedAmount       *big.Int
	Timestamp            int32
	SendHash             types.Hash
}