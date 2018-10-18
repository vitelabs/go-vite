package rpcapi

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
)

func Init(dir, lvl string, hexPrivkey string) {
	api.InitLog(dir, lvl)
	api.InitHexPrivKey(hexPrivkey)
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
	case "contracts":
		return rpc.API{
			Namespace: "contracts",
			Version:   "1.0",
			Service:   api.NewContractsApi(vite),
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
	return GetApis(vite, "ledger", "public_onroad", "net", "contracts", "testapi", "pow", "tx")
}

func GetAllApis(vite *vite.Vite) []rpc.API {
	return GetApis(vite, "ledger", "wallet", "private_onroad", "net", "contracts", "testapi", "pow", "tx")
}
