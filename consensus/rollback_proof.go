package consensus

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var NotFoundBlock = errors.New("block not found")

type RollbackProof interface {
	Proof(hash types.Hash, t time.Time) (*ledger.SnapshotBlock, error)
	ProofEmpty(stime time.Time, etime time.Time) (bool, error)
	ProofHash(t time.Time) (types.Hash, error)
}

type rollbackProof struct {
	rw Chain
}

func (self *rollbackProof) ProofEmpty(stime time.Time, etime time.Time) (bool, error) {
	block, err := self.rw.GetSnapshotHeaderBeforeTime(&etime)
	if err != nil {
		return false, err
	}
	if block == nil {
		return false, errors.New("before time[" + etime.String() + "] block not exist")
	}
	// todo
	//fmt.Printf("[%s]\t[%s]\t[%s], height:%d\n", stime, block.Timestamp, etime, block.Height)
	if block.Timestamp.Before(stime) {
		return true, nil
	} else {
		return false, nil
	}
}

func (self *rollbackProof) ProofHash(t time.Time) (types.Hash, error) {
	block, err := self.rw.GetSnapshotHeaderBeforeTime(&t)
	if err != nil {
		return types.Hash{}, err
	}
	if block == nil {
		return types.Hash{}, errors.New("before time[" + t.String() + "] block not exist")
	}
	return block.Hash, err
}

func (self *rollbackProof) Proof(hash types.Hash, t time.Time) (*ledger.SnapshotBlock, error) {
	block, err := self.rw.GetSnapshotHeaderBeforeTime(&t)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.New("before time[" + t.String() + "] block not exist")
	}

	if block.Hash != hash {
		return nil, errors.Errorf("block[%s][%s] proof fail.", hash, block.Hash)
	}
	return block, nil
}

func newRollbackProof(rw Chain) *rollbackProof {
	return &rollbackProof{rw: rw}
}
