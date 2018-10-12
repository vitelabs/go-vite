package rpcapi

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
)

func getApi(vite *vite.Vite, apiModule string) rpc.API {
	switch apiModule {
	case "ledger":
		return rpc.API{
			Namespace: "ledger",
			Version:   "1.0",
			Service:   api.NewLedgerApi(vite),
			Public:    true,
		}

	case "wallet":
		return rpc.API{
			Namespace: "wallet",
			Version:   "1.0",
			Service:   api.NewWalletApi(vite),
			Public:    true,
		}

	case "onroad":
		return rpc.API{
			Namespace: "onroad",
			Version:   "1.0",
			Service:   api.NewPrivateOnroadApi(vite.OnRoad()),
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
	case "testapi":
		return rpc.API{
			Namespace: "testapi",
			Version:   "1.0",
			Service:   api.NewTestApi(api.NewWalletApi(vite)),
			Public:    true,
		}
	case "pow":
		return rpc.API{
			Namespace: "pow",
			Version:   "1.0",
			Service:   api.Pow{},
			Public:    true,
		}
	default:
		return rpc.API{}
	}
}

func getApis(vite *vite.Vite, apiModule ...string) []rpc.API {
	var apis []rpc.API
	for _, m := range apiModule {
		apis = append(apis, getApi(vite, m))
	}
	return apis
}

func GetPublicApis(vite *vite.Vite) []rpc.API {
	return getApis(vite, "ledger", "wallet", "onroad", "net", "contracts", "testapi", "pow")
}

func GetAllApis(vite *vite.Vite) []rpc.API {
	return getApis(vite, "ledger", "wallet", "onroad", "net", "contracts", "testapi", "pow")
}
