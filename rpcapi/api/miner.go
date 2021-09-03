package api

import (
	"context"
	"errors"
	"time"

	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if api.vite.Config().IsMine() {
		addr := api.vite.Producer().GetCoinBase()
		from := api.chain.GetLatestSnapshotBlock().Height
		err := api.cs.TriggerMineEvent(addr)
		if err != nil {
			return err
		}

		// waiting for next snapshot block, with timeout 5s
		for {
			select {
			case <-ctx.Done():
				return errors.New("timeout for mine new block")
			default:
				to := api.chain.GetLatestSnapshotBlock().Height
				if to > from {
					return nil
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	return errors.New("should enable mine")
}
