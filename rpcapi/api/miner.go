package api

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
)

type MinerApi struct {
	vite  *vite.Vite
	chain chain.Chain
	cs    consensus.Consensus
}

func NewMinerApi(vite *vite.Vite) *MinerApi {
	return &MinerApi{
		vite:  vite,
		chain: vite.Chain(),
		cs:    vite.Consensus(),
	}
}

func (api MinerApi) String() string {
	return "MinerApi"
}

func (api *MinerApi) Mine() error {
	if !api.vite.Config().IsMine() {
		return fmt.Errorf("should enable mine")
	}
	if !api.vite.Config().ExternalMiner {
		return fmt.Errorf("should enable external miner")
	}
	return api.vite.Producer().SnapshotOnce()
}
