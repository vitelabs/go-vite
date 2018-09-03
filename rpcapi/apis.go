package rpcapi

import (
	"github.com/vitelabs/go-vite/rpcapi/api/impl"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/rpc"
)

func GetAllApis(vite *vite.Vite) []rpc.API {
	//ledgerApis := rpc.API{
	//	Namespace: "ledger",
	//	Version:   "1.0",
	//	Service:   impl.NewLedgerApi(vite),
	//	Public:    true,
	//}

	walletApis := rpc.API{
		Namespace: "wallet",
		Version:   "1.0",
		Service:   impl.NewWalletApi(vite),
		Public:    true,
	}

	//p2pApis := rpc.API{
	//	Namespace: "p2p",
	//	Version:   "1.0",
	//	Service:   impl.NewP2PApi(vite.P2p()),
	//	Public:    true,
	//}
	//
	//typesApis := rpc.API{
	//	Namespace: "types",
	//	Version:   "1.0",
	//	Service:   impl.TypesApisImpl{},
	//	Public:    true,
	//}
	//
	//commonApis := rpc.API{
	//	Namespace: "common",
	//	Version:   "1.0",
	//	Service:   impl.CommonApisImpl{},
	//	Public:    true,
	//}

	return []rpc.API{
		walletApis,
	}

	//return []rpc.API{
	//	ledgerApis,
	//	walletApis,
	//	p2pApis,
	//	typesApis,
	//	commonApis,
	//}
}
