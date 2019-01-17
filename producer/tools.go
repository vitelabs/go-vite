package producer

import (
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

func (self *tools) ledgerLock() {
	self.pool.Lock()
}
func (self *tools) ledgerUnLock() {
	self.pool.UnLock()
}

func (self *tools) generateSnapshot(e *consensus.Event, coinbase *AddressContext) (*ledger.SnapshotBlock, error) {
	head := self.chain.GetLatestSnapshotBlock()
	accounts, err := self.generateAccounts(head)
	if err != nil {
		return nil, err
	}
	trie, err := self.chain.GenStateTrie(head.StateHash, accounts)
	if err != nil {
		return nil, err
	}
	block := &ledger.SnapshotBlock{
		PrevHash:        head.Hash,
		Height:          head.Height + 1,
		Timestamp:       &e.Timestamp,
		StateTrie:       trie,
		StateHash:       *trie.Hash(),
		SnapshotContent: accounts,
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
	self.log.Info("insert snapshot block.", "block", block)
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

	needSnapshotAccounts := self.chain.GetNeedSnapshotContent()

	// todo get block
	for k, b := range needSnapshotAccounts {
		hashH, err := self.sVerifier.VerifyAccountTimeout(k, head.Height+1)
		if err != nil {
			self.log.Error("account verify timeout.", "addr", k, "accHash", b.Hash, "accHeight", b.Height, "err", err)
			if hashH != nil {
				err := self.pool.RollbackAccountTo(k, hashH.Hash, hashH.Height)
				if err != nil {
					self.log.Error("account rollback err1.", "addr", k, "accHash", hashH.Hash, "accHeight", hashH.Height, "err", err)
					return nil, err
				}
			} else {
				err := self.pool.RollbackAccountTo(k, b.Hash, b.Height)
				if err != nil {
					self.log.Error("account rollback err2.", "addr", k, "accHash", b.Hash, "accHeight", b.Height, "err", err)
					return nil, err
				}
			}
		}
	}

	// todo get block
	needSnapshotAccounts = self.chain.GetNeedSnapshotContent()

	var finalAccounts = make(map[types.Address]*ledger.HashHeight)

	for k, b := range needSnapshotAccounts {

		errB := b
		hashH, err := self.sVerifier.VerifyAccountTimeout(k, head.Height+1)
		if hashH != nil {
			errB = hashH
		}
		if err != nil {
			return nil, errors.Errorf(
				"error account block, account:%s, blockHash:%s, blockHeight:%d, err:%s",
				k.String(),
				errB.Hash.String(),
				errB.Height,
				err)
		}
		finalAccounts[k] = &ledger.HashHeight{Hash: b.Hash, Height: b.Height}
	}
	return finalAccounts, nil
}
