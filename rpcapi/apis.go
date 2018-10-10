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
	return getApis(vite, "ledger", "wallet", "onroad", "net")

	//
	//p2pApis := rpc.API{
	//	Namespace: "p2p",
	//	Version:   "1.0",
	//	Service:   api.NewP2PApi(vite.P2p()),
	//	Public:    true,
	//}
	//
	//typesApis := rpc.API{
	//	Namespace: "types",
	//	Version:   "1.0",
	//	Service:   api.TypesApi{},
	//	Public:    true,
	//}
	//
	//commonApis := rpc.API{
	//	Namespace: "common",
	//	Version:   "1.0",
	//	Service:   api.CommonApi{},
	//	Public:    true,
	//}
	//
	//return []rpc.API{
	//	ledgerApis,
	//	p2pApis,
	//	typesApis,
	//	commonApis,
	//}
}

func GetAllApis(vite *vite.Vite) []rpc.API {
	return getApis(vite, "ledger", "wallet", "onroad", "net")

	//walletApis := rpc.API{
	//	Namespace: "wallet",
	//	Version:   "1.0",
	//	Service:   api.NewWalletApi(vite),
	//	Public:    true,
	//}
	//
	//p2pApis := rpc.API{
	//	Namespace: "p2p",
	//	Version:   "1.0",
	//	Service:   api.NewP2PApi(vite.P2p()),
	//	Public:    true,
	//}
	//
	//typesApis := rpc.API{
	//	Namespace: "types",
	//	Version:   "1.0",
	//	Service:   api.TypesApi{},
	//	Public:    true,
	//}
	//
	//commonApis := rpc.API{
	//	Namespace: "common",
	//	Version:   "1.0",
	//	Service:   api.CommonApi{},
	//	Public:    true,
	//}
	//
	//return []rpc.API{
	//	ledgerApis,
	//	walletApis,
	//	p2pApis,
	//	typesApis,
	//	commonApis,
	//}
}
