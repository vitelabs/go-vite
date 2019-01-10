package rpcapi

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
)

func Init(dir, lvl string, testApi_prikey, testApi_tti string) {
	api.InitLog(dir, lvl)
	api.InitTestAPIParams(testApi_prikey, testApi_tti)
	api.InitGetTestTokenLimitPolicy()
}

func GetApi(vite *vite.Vite, apiModule string) rpc.API {
	switch apiModule {
	// private IPC
	case "wallet":
		return rpc.API{
			Namespace: "wallet",
			Version:   "1.0",
			Service:   api.NewWalletApi(vite),
			Public:    false,
		}
	case "private_onroad":
		return rpc.API{
			Namespace: "onroad",
			Version:   "1.0",
			Service:   api.NewPrivateOnroadApi(vite),
			Public:    false,
		}
		// public  WS HTTP IPC
	case "pow":
		return rpc.API{
			Namespace: "pow",
			Version:   "1.0",
			Service:   api.Pow{},
			Public:    true,
		}

	case "ledger":
		return rpc.API{
			Namespace: "ledger",
			Version:   "1.0",
			Service:   api.NewLedgerApi(vite),
			Public:    true,
		}
	case "public_onroad":
		return rpc.API{
			Namespace: "onroad",
			Version:   "1.0",
			Service:   api.NewPublicOnroadApi(vite),
			Public:    true,
		}
	case "net":
		return rpc.API{
			Namespace: "net",
			Version:   "1.0",
			Service:   api.NewNetApi(vite),
			Public:    true,
		}
	case "contract":
		return rpc.API{
			Namespace: "contract",
			Version:   "1.0",
			Service:   api.NewContractApi(vite),
			Public:    true,
		}
	case "register":
		return rpc.API{
			Namespace: "register",
			Version:   "1.0",
			Service:   api.NewRegisterApi(vite),
			Public:    true,
		}
	case "vote":
		return rpc.API{
			Namespace: "vote",
			Version:   "1.0",
			Service:   api.NewVoteApi(vite),
			Public:    true,
		}
	case "mintage":
		return rpc.API{
			Namespace: "mintage",
			Version:   "1.0",
			Service:   api.NewMintageApi(vite),
			Public:    true,
		}
	case "pledge":
		return rpc.API{
			Namespace: "pledge",
			Version:   "1.0",
			Service:   api.NewPledgeApi(vite),
			Public:    true,
		}
	case "consensusGroup":
		return rpc.API{
			Namespace: "consensusGroup",
			Version:   "1.0",
			Service:   api.NewConsensusGroupApi(vite),
			Public:    true,
		}
	case "tx":
		return rpc.API{
			Namespace: "tx",
			Version:   "1.0",
			Service:   api.NewTxApi(vite),
			Public:    true,
		}
		// test
	case "testapi":
		return rpc.API{
			Namespace: "testapi",
			Version:   "1.0",
			Service:   api.NewTestApi(api.NewWalletApi(vite)),
			Public:    true,
		}
	case "debug":
		return rpc.API{
			Namespace: "debug",
			Version:   "1.0",
			Service:   api.NewDebugApi(vite),
			Public:    true,
		}
	case "dashboard":
		return rpc.API{
			Namespace: "dashboard",
			Version:   "1.0",
			Service:   api.NewDashboardApi(vite),
			Public:    true,
		}
	case "vmdebug":
		return rpc.API{
			Namespace: "vmdebug",
			Version:   "1.0",
			Service:   api.NewVmDebugApi(vite),
			Public:    true,
		}
	default:
		return rpc.API{}
	}
}

func GetApis(vite *vite.Vite, apiModule ...string) []rpc.API {
	var apis []rpc.API
	for _, m := range apiModule {
		apis = append(apis, GetApi(vite, m))
	}
	return apis
}

func GetPublicApis(vite *vite.Vite) []rpc.API {
	return GetApis(vite, "ledger", "public_onroad", "net", "contract", "pledge", "register", "vote", "mintage", "consensusGroup", "testapi", "pow", "tx", "debug", "dashboard")
}

func GetAllApis(vite *vite.Vite) []rpc.API {
	return GetApis(vite, "ledger", "wallet", "private_onroad", "net", "contract", "pledge", "register", "vote", "mintage", "consensusGroup", "testapi", "pow", "tx", "debug", "dashboard", "vmdebug")
}
