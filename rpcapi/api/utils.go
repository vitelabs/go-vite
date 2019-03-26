package api

import (
	"context"
	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	log                              = log15.New("module", "rpc/api")
	testapi_hexPrivKey               = ""
	testapi_tti                      = ""
	convertError                     = errors.New("convert error")
	testapi_testtokenlru  *lru.Cache = nil
	testtokenlruCron      *cron.Cron = nil
	testtokenlruLimitSize            = 20
	dataDir                          = ""
	netId                            = uint(0)
)

func InitConfig(id uint) {
	netId = id
}

func InitLog(dir, lvl string) {
	dataDir = dir
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

func InitGetTestTokenLimitPolicy() {
	testapi_testtokenlru, _ = lru.New(2048)

	testtokenlruCron = cron.New()
	testtokenlruCron.AddFunc("@daily", func() {
		log.Info("clear lrucache")
		if testapi_testtokenlru != nil {
			newcache, _ := lru.New(2048)
			testapi_testtokenlru = newcache
		}
	})

	testtokenlruCron.Start()
}

func CheckGetTestTokenIpFrequency(cache *lru.Cache, ctx context.Context) error {
	if cache == nil {
		return nil
	}
	endpoint, ok := ctx.Value("remote").(string)
	if ok {
		log.Info("GetTestToken", "remote", endpoint)
		split := strings.Split(endpoint, ":")
		if len(split) == 2 {
			ip := split[0]
			if count, ok := cache.Get(ip); ok {
				c := count.(int)
				if c >= testtokenlruLimitSize {
					return errors.New("too frequent")
				} else {
					c++
					cache.Add(ip, c)
				}
			} else {
				cache.Add(ip, 1)
			}
		}
	}
	return nil
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

func StringToUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

const (
	secondBetweenSnapshotBlocks int64 = 1
)

func getWithdrawTime(snapshotTime *time.Time, snapshotHeight uint64, withdrawHeight uint64) int64 {
	return snapshotTime.Unix() + int64(withdrawHeight-snapshotHeight)*secondBetweenSnapshotBlocks
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

func getPrevBlockHash(c chain.Chain, addr *types.Address) (*types.Hash, error) {
	b, err := c.GetLatestAccountBlock(addr)
	if err != nil {
		return nil, err
	}
	if b != nil {
		return &b.Hash, nil
	}
	return &types.Hash{}, nil
}
