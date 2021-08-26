package producer

import (
	"time"

	"github.com/pkg/errors"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/common/upgrade"
	"github.com/vitelabs/go-vite/interfaces"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
	"github.com/vitelabs/go-vite/ledger/pool"
	"github.com/vitelabs/go-vite/ledger/verifier"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
)

type tools struct {
	log       log15.Logger
	pool      pool.SnapshotProducerWriter
	chain     chain.Chain
	sVerifier *verifier.SnapshotVerifier
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
		block.Version = upgrade.GetCurPoint(block.Height).Version
	}

	block.Hash = block.ComputeHash()

	signedData, pubkey, err := coinbase.Sign(block.Hash)

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

func newChainRw(ch chain.Chain, sVerifier *verifier.SnapshotVerifier, p pool.SnapshotProducerWriter) *tools {
	log := log15.New("module", "tools")
	return &tools{chain: ch, log: log, sVerifier: sVerifier, pool: p}
}

func (self *tools) generateAccounts(head *ledger.SnapshotBlock) (ledger.SnapshotContent, error) {
	// todo get block
	needSnapshotAccounts := self.chain.GetContentNeedSnapshot()

	//var finalAccounts = make(map[types.Address]*ledger.HashHeight)

	//for k, b := range needSnapshotAccounts {
	//
	//	errB := b
	//	hashH, err := self.sVerifier.VerifyAccountTimeout(k, b, head.Height+1)
	//	if hashH != nil {
	//		errB = hashH
	//	}
	//	if err != nil {
	//		return nil, errors.Errorf(
	//			"error account block, account:%s, blockHash:%s, blockHeight:%d, err:%s",
	//			k.String(),
	//			errB.Hash.String(),
	//			errB.Height,
	//			err)
	//	}
	//	finalAccounts[k] = &ledger.HashHeight{Hash: b.Hash, Height: b.Height}
	//}
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
