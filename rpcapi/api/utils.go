package api

import (
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

var (
	log        = log15.New("module", "rpc/api")
	hexPrivKey = ""
)

func InitLog(dir, lvl string) {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	filename := time.Now().Format("2006-01-02") + ".log"
	path := filepath.Join(dir, "rpclog")
	if err := os.MkdirAll(path, 0777); err != nil {
		return
	}
	absFilename := filepath.Join(path, filename)
	log.SetHandler(
		log15.LvlFilterHandler(logLevel, log15.Must.FileHandler(absFilename, log15.TerminalFormat())),
	)
}

func InitHexPrivKey(priv string) {
	hexPrivKey = priv
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
