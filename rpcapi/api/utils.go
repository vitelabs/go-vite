package api

import (
	"math/big"
	"strconv"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
)

var (
	log                              = log15.New("module", "rpc/api")
	testapi_hexPrivKey               = ""
	testapi_tti                      = ""
	convertError                     = errors.New("convert error")
	testapi_testtokenlru  *lru.Cache = nil
	testtokenlruLimitSize            = 20
	dataDir                          = ""
	netId                            = uint(0)
	dexTxAvailable                   = false
)

func InitConfig(id uint, dexAvailable *bool) {
	netId = id
	if dexAvailable != nil && *dexAvailable {
		dexTxAvailable = *dexAvailable
	}
}

func InitLog(dir, lvl string) {
	log.SetHandler(common.LogHandler(dir, "rpclog", "rpc.log", lvl))
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

func Uint64ToString(u uint64) string {
	return strconv.FormatUint(u, 10)
}

func StringToUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func Float64ToString(f float64, prec int) string {
	return strconv.FormatFloat(f, 'g', prec, 64)
}
func StringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

const (
	secondBetweenSnapshotBlocks int64 = 1
)

func getWithdrawTime(snapshotTime *time.Time, snapshotHeight uint64, expirationHeight uint64) int64 {
	return snapshotTime.Unix() + int64(expirationHeight-snapshotHeight)*secondBetweenSnapshotBlocks
}

func getRange(index, count, listLen int) (int, int) {
	start := index * count
	if start >= listLen {
		return listLen, listLen
	}
	end := start + count
	if end >= listLen {
		return start, listLen
	}
	return start, end
}

func getPrevBlockHash(c chain.Chain, addr types.Address) (*types.Hash, error) {
	b, err := c.GetLatestAccountBlock(addr)
	if err != nil {
		return nil, err
	}
	if b != nil {
		return &b.Hash, nil
	}
	return &types.Hash{}, nil
}

func getVmDb(c chain.Chain, addr types.Address) (interfaces.VmDb, error) {
	prevHash, err := getPrevBlockHash(c, addr)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(c, &addr, &c.GetLatestSnapshotBlock().Hash, prevHash)
	return db, err
}

func checkTxToAddressAvailable(address types.Address) bool {
	if !dexTxAvailable {
		return address != types.AddressDexTrade && address != types.AddressDexFund
	}
	return true
}

func checkSnapshotValid(latestSb *ledger.SnapshotBlock) error {
	nowTime := time.Now()
	if nowTime.Before(latestSb.Timestamp.Add(-10*time.Minute)) || nowTime.After(latestSb.Timestamp.Add(10*time.Minute)) {
		return IllegalNodeTime
	}
	return nil
}

func checkTokenIdValid(chain chain.Chain, tokenId *types.TokenTypeId) error {
	if tokenId != nil && (*tokenId) != types.ZERO_TOKENID {
		tkInfo, err := chain.GetTokenInfoById(*tokenId)
		if err != nil {
			return err
		}
		if tkInfo == nil {
			return errors.New("tokenId doesn’t exist")
		}
	}
	return nil
}
