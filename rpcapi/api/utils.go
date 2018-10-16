package api

import (
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

var log = log15.New("module", "rpc/api")

func InitLog(dir, lvl string) {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	log15.Root().SetHandler(
		log15.LvlFilterHandler(logLevel, log15.Must.FileHandler(dir, log15.TerminalFormat())),
	)
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
