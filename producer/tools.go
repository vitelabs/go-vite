package producer

import (
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type tools struct {
	log log15.Logger
}

func (self *tools) lock() {
	self.log.Info("lock")
}
func (self *tools) unLock() {
	self.log.Info("unlock")
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
