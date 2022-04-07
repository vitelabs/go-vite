package vite

import (
	"github.com/vitelabs/go-vite/v2/common/config"
	nodeconfig "github.com/vitelabs/go-vite/v2/node/config"
	"github.com/vitelabs/go-vite/v2/wallet"
)

func NewMock(cfg *config.Config, walletManager *wallet.Manager) (vite *Vite, err error) {
	var nodeConfig *nodeconfig.Config
	if cfg == nil || walletManager == nil {
		nodeConfig = &nodeconfig.Config{}
		//nodeConfig.ParseFromFile("~/go/src/github.com/vitelabs/go-vite/conf/evm/node_config.json")
	}
	if cfg == nil {
		cfg = nodeConfig.MakeViteConfig()
	}
	if walletManager == nil {
		walletManager = wallet.New(nodeConfig.MakeWalletConfig())
	}

	// TODO: use mocks for net, chain, pool, etc.

	vite = &Vite{
		config:        cfg,
		walletManager: walletManager,
		net:           nil,
		chain:         nil,
		pool:          nil,
		consensus:     nil,
		verifier:      nil,
	}
	return
}
