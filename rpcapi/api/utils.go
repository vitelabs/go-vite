package api

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

var log = log15.New("module", "rpc/api")

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}

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
