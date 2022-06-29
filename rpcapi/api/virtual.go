package api

import (
	"errors"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
)

type VirtualApi struct {
	vite  *vite.Vite
	chain chain.Chain
	cs    consensus.Consensus
}

func NewVirtualApi(vite *vite.Vite) *VirtualApi {
	return &VirtualApi{
		vite:  vite,
		chain: vite.Chain(),
		cs:    vite.Consensus(),
	}
}

func (api VirtualApi) String() string {
	return "VirtualApi"
}

func (api *VirtualApi) Mine() error {
	return api.vite.Producer().SnapshotOnce()
}

func (api *VirtualApi) MineBatch(number uint64) error {
	if number > 1000 {
		return errors.New("number must be less than 1000")
	}

	for i := uint64(0); i < number; i++ {
		err := api.vite.Producer().SnapshotOnce()
		if err != nil {
			return err
		}
	}
	return nil
}

func (api *VirtualApi) AddUpgrade(version uint32, height uint64) error {
	return upgrade.AddUpgradePoint(version, height)
}
