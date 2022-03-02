package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/log15"
)

type dposReader struct {
	snapshot  DposReader
	contracts *contractsCs

	log log15.Logger
}

func (self *dposReader) getDposConsensus(gid types.Gid) (DposReader, error) {
	if gid == types.SNAPSHOT_GID {
		return self.snapshot, nil
	}

	return self.contracts.getOrLoadGid(gid)
}

type DposReader interface {
	ElectionIndex(index uint64) (*electionResult, error)
	GetInfo() *core.GroupInfo
	Time2Index(t time.Time) uint64
	Index2Time(i uint64) (time.Time, time.Time)
	GenProofTime(t uint64) time.Time
	VerifyProducer(address types.Address, t time.Time) (bool, error)
}

type DposVerifier interface {
}
