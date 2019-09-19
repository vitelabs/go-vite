package producer

import (
	"github.com/vitelabs/go-vite/common/fork"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

type tools struct {
	log       log15.Logger
	wt        *wallet.Manager
	pool      pool.SnapshotProducerWriter
	chain     chain.Chain
	sVerifier *verifier.SnapshotVerifier
}

func (self *tools) generateSnapshot(e *consensus.Event, coinbase *AddressContext, seed uint64, fn func(*types.Hash) uint64) (*ledger.SnapshotBlock, error) {
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
	if fork.IsLeafFork(block.Height) {
		block.Version = fork.GetLastForkPoint().Version
	}

	block.Hash = block.ComputeHash()
	manager, err := self.wt.GetEntropyStoreManager(coinbase.EntryPath)
	if err != nil {
		return nil, err
	}
	_, key, err := manager.DeriveForIndexPath(coinbase.Index)
	if err != nil {
		return nil, err
	}
	signedData, pubkey, err := key.SignData(block.Hash.Bytes())

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

func newChainRw(ch chain.Chain, sVerifier *verifier.SnapshotVerifier, wt *wallet.Manager, p pool.SnapshotProducerWriter) *tools {
	log := log15.New("module", "tools")
	return &tools{chain: ch, log: log, sVerifier: sVerifier, wt: wt, pool: p}
}

func (self *tools) checkAddressLock(address types.Address, coinbase *AddressContext) error {
	if address != coinbase.Address {
		return errors.Errorf("addres not equals.%s-%s", address, coinbase.Address)
	}

	return self.wt.MatchAddress(coinbase.EntryPath, coinbase.Address, coinbase.Index)
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
