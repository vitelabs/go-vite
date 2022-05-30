package producer

import (
	"time"

	"github.com/pkg/errors"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
	"github.com/vitelabs/go-vite/v2/ledger/pool"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/monitor"
)

type tools struct {
	log   log15.Logger
	pool  pool.SnapshotProducerWriter
	chain chain.Chain
}

func (self *tools) generateSnapshot(e *consensus.Event, coinbase interfaces.Account, seed uint64, fn func(*types.Hash) uint64) (*ledger.SnapshotBlock, error) {
	head := self.chain.GetLatestSnapshotBlock()
	accounts, err := self.generateAccounts(head)
	if err != nil {
		return nil, err
	}

	block := &ledger.SnapshotBlock{
		PrevHash:        head.Hash,
		Height:          head.Height + 1,
		Timestamp:       &e.Timestamp,
		SnapshotContent: accounts,
	}
	lastBlock := self.getLastSeedBlock(e, head, e.PeriodStime)
	if lastBlock != nil {
		if lastBlock.Timestamp.Before(e.PeriodStime) {
			lastSeed := fn(lastBlock.SeedHash)
			seedHash := ledger.ComputeSeedHash(seed, block.PrevHash, block.Timestamp)
			block.SeedHash = &seedHash
			block.Seed = lastSeed
		}
	} else {
		seedHash := ledger.ComputeSeedHash(seed, block.PrevHash, block.Timestamp)
		block.SeedHash = &seedHash
		block.Seed = 0
	}

	// add version
	if upgrade.IsLeafUpgrade(block.Height) {
		block.Version = upgrade.GetLatestPoint().Version
	}

	block.Hash = block.ComputeHash()

	signedData, pubkey, err := coinbase.Sign(block.Hash.Bytes())

	if err != nil {
		return nil, err
	}
	block.Signature = signedData
	block.PublicKey = pubkey
	return block, nil
}
func (self *tools) insertSnapshot(block *ledger.SnapshotBlock) error {
	defer monitor.LogTime("producer", "snapshotInsert", time.Now())
	// todo insert pool ?? dead lock
	self.log.Info("insert snapshot block.", "block", block, "producer", block.Producer())
	return self.pool.AddDirectSnapshotBlock(block)
}

func newChainRw(ch chain.Chain, p pool.SnapshotProducerWriter) *tools {
	log := log15.New("module", "tools")
	return &tools{chain: ch, log: log, pool: p}
}

func (self *tools) generateAccounts(head *ledger.SnapshotBlock) (ledger.SnapshotContent, error) {
	// todo get block
	needSnapshotAccounts := self.chain.GetContentNeedSnapshot()
	return needSnapshotAccounts, nil
}

func (self *tools) getLastSeedBlock(e *consensus.Event, head *ledger.SnapshotBlock, beforeTime time.Time) *ledger.SnapshotBlock {
	block, err := self.chain.GetLastUnpublishedSeedSnapshotHeader(e.Address, beforeTime)
	if err != nil {
		return nil
	}
	latest := self.chain.GetLatestSnapshotBlock()
	if latest.Hash != head.Hash {
		return nil
	}
	return block
}

func (self *tools) checkStableSnapshotChain(header *ledger.SnapshotBlock) error {
	if header.Height <= uint64(time.Minute) {
		return nil
	}
	targetH := header.Height - uint64(time.Minute)

	block, err := self.chain.GetSnapshotHeaderByHeight(targetH)
	if err != nil {
		return err
	}
	if block == nil {
		return errors.Errorf("empty snapshot block[%d]", targetH)
	}

	t := time.Now().Add(-time.Minute).Add(-time.Second * 20)
	if block.Timestamp.Before(t) {
		return errors.Errorf("snapshot is not stable[%s],[%s]", t, block.Timestamp)
	}
	return nil
}
