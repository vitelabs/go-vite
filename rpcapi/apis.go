package rpcapi

import (
	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/rpc"
	"github.com/vitelabs/go-vite/v2/rpcapi/api"
	"github.com/vitelabs/go-vite/v2/rpcapi/api/filters"
)

type ApiType uint

const (
	HEALTH = iota
	WALLET
	PRIVATE_ONROAD
	POW
	DEBUG
	CONSENSUSGROUP
	LEDGER
	PUBLIC_ONROAD
	NET
	CONTRACT
	REGISTER
	VOTE
	MINTAGE
	PLEDGE
	DEXFUND
	DEXTRADE
	DEX
	PRIVATE_DEX
	TX
	DASHBOARD
	SUBSCRIBE
	SBPSTATS
	UTIL
	DATA
	LEDGERDEBUG
	VIRTUAL
	apiTypeLimit // this will be the last ApiType + 1
)

var apiTypeStrings = []string{
	"health",
	"wallet",
	"private_onroad",
	"pow",
	"debug",
	"consensusGroup",
	"ledger",
	"public_onroad",
	"net",
	"contract",
	"register",
	"vote",
	"mintage",
	"pledge",
	"dexfund",
	"dextrade",
	"dex",
	"private_dex",
	"tx",
	"dashboard",
	"subscribe",
	"sbpstats",
	"util",
	"data",
	"ledgerdebug",
	"virtual",
}

func (at ApiType) name() string {
	return apiTypeStrings[at]
}

func (at ApiType) ordinal() int {
	return int(at)
}

func (at ApiType) values() *[]string {
	return &apiTypeStrings
}

func Init(dir, lvl string, testApi_prikey, testApi_tti string, netId uint, dexAvailable *bool) {
	api.InitLog(dir, lvl)
	api.InitTestAPIParams(testApi_prikey, testApi_tti)
	api.InitConfig(netId, dexAvailable)
}

func GetApi(vite *vite.Vite, apiModule string) rpc.API {
	switch apiModule {
	// private IPC
	case ApiType(HEALTH).name():
		return rpc.API{
			Namespace: "health",
			Version:   "1.0",
			Service:   api.NewHealthApi(vite),
			Public:    true,
		}
	case ApiType(WALLET).name():
		return rpc.API{
			Namespace: "wallet",
			Version:   "1.0",
			Service:   api.NewWalletApi(vite),
			Public:    false,
		}
	case ApiType(PRIVATE_ONROAD).name():
		return rpc.API{
			Namespace: "onroad",
			Version:   "1.0",
			Service:   api.NewPrivateOnroadApi(vite),
			Public:    false,
		}
	// public WS HTTP IPC
	case ApiType(POW).name():
		return rpc.API{
			Namespace: "pow",
			Version:   "1.0",
			Service:   api.NewPow(vite),
			Public:    true,
		}
	case ApiType(DEBUG).name():
		return rpc.API{
			Namespace: "debug",
			Version:   "1.0",
			Service:   api.NewDeprecated(),
			Public:    true,
		}
	case ApiType(CONSENSUSGROUP).name():
		return rpc.API{
			Namespace: "debug",
			Version:   "1.0",
			Service:   api.NewDeprecated(),
			Public:    true,
		}
	case ApiType(LEDGER).name():
		return rpc.API{
			Namespace: "ledger",
			Version:   "1.0",
			Service:   api.NewLedgerApi(vite),
			Public:    true,
		}
	case ApiType(PUBLIC_ONROAD).name():
		return rpc.API{
			Namespace: "onroad",
			Version:   "1.0",
			Service:   api.NewPublicOnroadApi(vite),
			Public:    true,
		}
	case ApiType(NET).name():
		return rpc.API{
			Namespace: "net",
			Version:   "1.0",
			Service:   api.NewNetApi(vite),
			Public:    true,
		}
	case ApiType(CONTRACT).name():
		return rpc.API{
			Namespace: "contract",
			Version:   "1.0",
			Service:   api.NewContractApi(vite),
			Public:    true,
		}
	case ApiType(REGISTER).name():
		return rpc.API{
			Namespace: "register",
			Version:   "1.0",
			Service:   api.NewRegisterApi(vite),
			Public:    true,
		}
	case ApiType(VOTE).name():
		return rpc.API{
			Namespace: "vote",
			Version:   "1.0",
			Service:   api.NewVoteApi(vite),
			Public:    true,
		}
	case ApiType(MINTAGE).name():
		return rpc.API{
			Namespace: "mintage",
			Version:   "1.0",
			Service:   api.NewMintageAPI(vite),
			Public:    true,
		}
	case ApiType(PLEDGE).name():
		return rpc.API{
			Namespace: "pledge",
			Version:   "1.0",
			Service:   api.NewQuotaApi(vite),
			Public:    true,
		}
	case ApiType(DEXFUND).name():
		return rpc.API{
			Namespace: "dexfund",
			Version:   "1.0",
			Service:   api.NewDexFundApi(vite),
			Public:    true,
		}
	case ApiType(DEXTRADE).name():
		return rpc.API{
			Namespace: "dextrade",
			Version:   "1.0",
			Service:   api.NewDexTradeApi(vite),
			Public:    true,
		}
	case ApiType(DEX).name():
		return rpc.API{
			Namespace: "dex",
			Version:   "1.0",
			Service:   api.NewDexApi(vite),
			Public:    true,
		}
	case ApiType(PRIVATE_DEX).name():
		return rpc.API{
			Namespace: "dex",
			Version:   "1.0",
			Service:   api.NewDexPrivateApi(vite),
			Public:    false,
		}
	case ApiType(TX).name():
		return rpc.API{
			Namespace: "tx",
			Version:   "1.0",
			Service:   api.NewTxApi(vite),
			Public:    true,
		}
	case ApiType(DASHBOARD).name():
		return rpc.API{
			Namespace: "dashboard",
			Version:   "1.0",
			Service:   api.NewDashboardApi(vite),
			Public:    true,
		}
	case ApiType(SUBSCRIBE).name():
		return rpc.API{
			Namespace: "subscribe",
			Version:   "1.0",
			Service:   filters.NewSubscribeApi(vite),
			Public:    true,
		}
	case ApiType(SBPSTATS).name():
		return rpc.API{
			Namespace: "sbpstats",
			Version:   "1.0",
			Service:   api.NewStatsApi(vite),
			Public:    true,
		}
	case ApiType(UTIL).name():
		return rpc.API{
			Namespace: "util",
			Version:   "1.0",
			Service:   api.NewUtilApi(vite),
			Public:    true,
		}
	case ApiType(DATA).name():
		return rpc.API{
			Namespace: "data",
			Version:   "1.0",
			Service:   api.NewDataApi(vite),
			Public:    true,
		}
	case ApiType(LEDGERDEBUG).name():
		return rpc.API{
			Namespace: "ledgerdebug",
			Version:   "1.0",
			Service:   api.NewLedgerDebugApi(vite),
			Public:    false,
		}
	case ApiType(VIRTUAL).name():
		return rpc.API{
			Namespace: "virtual",
			Version:   "1.0",
			Service:   api.NewVirtualApi(vite),
			Public:    false,
		}
	default:
		return rpc.API{Namespace: apiModule}
	}
}

func GetApis(vite *vite.Vite, apiModules ...string) map[string]rpc.API {
	var apis = make(map[string]rpc.API, len(apiModules))
	for _, m := range apiModules {
		apis[m] = GetApi(vite, m)
	}
	return apis
}
func MergeApis(first map[string]rpc.API, second map[string]rpc.API) []rpc.API {
	resultMap := make(map[string]rpc.API)

	for md, api := range first {
		resultMap[md] = api
	}

	for md1, api1 := range second {
		if _, ok := resultMap[md1]; ok {
			continue
		} else {
			resultMap[md1] = api1
		}
	}

	var result []rpc.API
	for _, r := range resultMap {
		result = append(result, r)
	}
	return result
}

func GetPublicApis(vite *vite.Vite) map[string]rpc.API {
	return GetApis(vite, ApiType(LEDGER).name(), ApiType(NET).name(), ApiType(CONTRACT).name(), ApiType(UTIL).name(), ApiType(HEALTH).name())
}
