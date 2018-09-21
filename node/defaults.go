package node

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
)

var DefaultNodeConfig = Config{
	DataDir:  common.DefaultDataDir(),
	HttpPort: common.DefaultHTTPPort,
	WSPort:   common.DefaultWSPort,
	P2P: config.P2P{
		Name:            "vite-server",
		PrivateKey:      "",
		MaxPeers:        100,
		MaxInboundRatio: 2,
		MaxPendingPeers: 20,
		BootNodes:       nil,
		Port:            8483,
		Datadir:         common.DefaultDataDir(),
		NetID:           6,
	},
}
