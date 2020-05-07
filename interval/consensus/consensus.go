package consensus

import (
	"github.com/vitelabs/go-vite/interval/common"
)

type SnapshotReader interface {
}

type SnapshotHeader struct {
	Timestamp uint64
	Producer  common.Address
}

type ConsensusVerifier interface {
	Verify(reader SnapshotReader, block *common.SnapshotBlock) (bool, error)
}

type Seal interface {
	Seal() error
}

type AccountsConsensus interface {
	//ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error
	//ForkAccountTo(h *common.AccountHashH) error
	//SnapshotAccount(block *common.SnapshotBlock, h *common.AccountHashH)
	//UnLockAccounts(startAcs map[string]*common.SnapshotPoint, endAcs map[string]*common.SnapshotPoint) error
	//PendingAccountTo(h *common.AccountHashH) error
}

type Consensus interface {
	ConsensusVerifier
	Seal

	Subscribe(subscribeMem *SubscribeMem)
	Init()
	Start()
	Stop()
}
