package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"path/filepath"
	"strconv"
	"time"
)

var (
	log                = log15.New("module", "rpc/api")
	testapi_hexPrivKey = ""
	testapi_tti        = ""
	convertError       = errors.New("convert error")
)

func InitLog(dir, lvl string) {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	path := filepath.Join(dir, "rpclog", time.Now().Format("2006-01-02T15-04"))
	filename := filepath.Join(path, "rpc.log")
	log.SetHandler(
		log15.LvlFilterHandler(logLevel, log15.StreamHandler(common.MakeDefaultLogger(filename), log15.LogfmtFormat())),
	)
}

func InitTestAPIParams(priv, tti string) {
	testapi_hexPrivKey = priv
	testapi_tti = tti
}

func stringToBigInt(str *string) (*big.Int, error) {
	if str == nil {
		return nil, convertError
	}
	n := new(big.Int)
	n, ok := n.SetString(*str, 10)
	if n == nil || !ok {
		return nil, convertError
	}
	return n, nil
}

func bigIntToString(big *big.Int) *string {
	if big == nil {
		return nil
	}
	s := big.String()
	return &s
}

func uint64ToString(u uint64) string {
	return strconv.FormatUint(u, 10)
}

const (
	secondBetweenSnapshotBlocks int64 = 1
)

func getWithdrawTime(snapshotTime *time.Time, snapshotHeight uint64, withdrawHeight uint64) int64 {
	return snapshotTime.Unix() + int64(withdrawHeight-snapshotHeight)*secondBetweenSnapshotBlocks
}
