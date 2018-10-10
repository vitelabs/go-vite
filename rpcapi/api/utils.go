package api

import (
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

var log = log15.New("module", "rpc/api")


func stringToBigInt(str *string) *big.Int {
	if str == nil {
		return nil
	}
	n := new(big.Int)
	n, ok := n.SetString(*str, 10)
	if n == nil || !ok {
		return nil
	}
	return n
}

func bigIntToString(big *big.Int) *string {
	if big == nil {
		return nil
	}
	s := big.String()
	return &s
}
