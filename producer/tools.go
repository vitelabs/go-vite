package producer

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/wallet"
)

type tools struct {
	log  log15.Logger
	wt   wallet.Manager
	pool pool.PoolWriter
}

func (self *tools) ledgerLock() {
	self.pool.Lock()
}
func (self *tools) ledgerUnLock() {
	self.pool.UnLock()
}
func (self *tools) generateSnapshot(e *consensus.Event) *ledger.SnapshotBlock {
	self.log.Info("generate")
	return nil
}
func (self *tools) insertSnapshot(block *ledger.SnapshotBlock) {
	self.log.Info("insert")
}

func newChainRw() *tools {
	log := log15.New("module", "tools")
	return &tools{log: log}
}

func (self *tools) checkAddressLock(address types.Address) bool {
	unLocked := self.wt.KeystoreManager.IsUnLocked(address)
	return unLocked
}
