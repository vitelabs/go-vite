package apis

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/vite"
)

func GetAll(vite *vite.Vite) []rpc.API {
	ledgerApis := rpc.API{
		Namespace: "ledger",
		Version:   "1.0",
		Service:   NewLedgerApi(vite),
		Public:    true,
	}

	walletApis := rpc.API{
		Namespace: "wallet",
		Version:   "1.0",
		Service:   NewWalletApi(vite),
		Public:    true,
	}

	p2pApis := rpc.API{
		Namespace: "p2p",
		Version:   "1.0",
		Service:   NewP2PApi(vite.P2p()),
		Public:    true,
	}

	return []rpc.API{
		ledgerApis,
		walletApis,
		p2pApis,
	}
}
