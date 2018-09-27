package node

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
)

var DefaultNodeConfig = Config{
	Name:     "gvite-config",
	IPCPath:  "vite.ipc",
	DataDir:  common.DefaultDataDir(),
	HttpPort: common.DefaultHTTPPort,
	WSPort:   common.DefaultWSPort,
	P2P: config.P2P{
		Name:                 "vite-server",
		PrivateKey:           "",
		MaxPeers:             100,
		MaxPassivePeersRatio: 2,
		MaxPendingPeers:      20,
		BootNodes:            nil,
		Port:                 8483,
		Datadir:              common.DefaultDataDir(),
		NetID:                6,
	},
}
