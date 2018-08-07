package rpc

import (
	"github.com/vitelabs/go-vite/rpc/api/impl"
	"github.com/vitelabs/go-vite/vite"
)

func GetAllApis(vite *vite.Vite) []API {
	ledgerApis := API{
		Namespace: "ledger",
		Version:   "1.0",
		Service:   impl.NewLedgerApi(vite),
		Public:    true,
	}

	walletApis := API{
		Namespace: "wallet",
		Version:   "1.0",
		Service:   impl.NewWalletApi(vite),
		Public:    true,
	}

	p2pApis := API{
		Namespace: "p2p",
		Version:   "1.0",
		Service:   impl.NewP2PApi(vite.P2p()),
		Public:    true,
	}

	typesApis := API{
		Namespace: "types",
		Version:   "1.0",
		Service:   impl.TypesApisImpl{},
		Public:    true,
	}

	commonApis := API{
		Namespace: "common",
		Version:   "1.0",
		Service:   impl.CommonApisImpl{},
		Public:    true,
	}

	return []API{
		ledgerApis,
		walletApis,
		p2pApis,
		typesApis,
		commonApis,
	}
}
