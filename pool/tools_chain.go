package pool

import (
	"strconv"

	"github.com/pkg/errors"
	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	//"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
)

type chainRw interface {
	insertBlock(block commonBlock) error
	insertBlocks(blocks []commonBlock) error

	head() commonBlock
	getBlock(height uint64) commonBlock
}

type accountCh struct {
	address types.Address
	rw      ch.Chain
	version *ForkVersion
}

func (self *accountCh) insertBlock(b commonBlock) error {
	if !b.checkForkVersion() {
		return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.FormatInt(int64(b.forkVersion()), 10))
	}
	block := b.(*accountPoolBlock)
	accountBlock := &vm_context.VmAccountBlock{AccountBlock: block.block, VmContext: block.vmBlock}
	return self.rw.InsertAccountBlocks([]*vm_context.VmAccountBlock{accountBlock})
}

func (self *accountCh) head() commonBlock {
	block, e := self.rw.GetLatestAccountBlock(&self.address)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}

	result := newAccountPoolBlock(block, nil, self.version)
	return result
}

func (self *accountCh) getBlock(height uint64) commonBlock {
	// todo
	return nil
}

func (self *accountCh) insertBlocks(bs []commonBlock) error {
	var blocks []*vm_context.VmAccountBlock
	for _, b := range bs {
		block := b.(*accountPoolBlock)
		blocks = append(blocks, &vm_context.VmAccountBlock{AccountBlock: block.block, VmContext: block.vmBlock})
	}

	return self.rw.InsertAccountBlocks(blocks)
}

func (self *accountCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	bm, e := self.rw.DeleteAccountBlocks(&self.address, height)
	if e != nil {
		return nil, nil, e
	}

	var results map[types.Address][]commonBlock
	for addr, bs := range bm {
		var r []commonBlock
		for _, b := range bs {
			r = append(r, newAccountPoolBlock(b, nil, self.version))
		}
		results[addr] = r
	}
	return nil, results, nil
}

func (self *accountCh) getUnConfirmedBlocks() []*ledger.AccountBlock {
	return self.rw.GetUnConfirmAccountBlocks(&self.address)
}
func (self *accountCh) getFirstUnconfirmedBlock(head *ledger.SnapshotBlock) *ledger.AccountBlock {
	block, e := self.rw.GetFirstConfirmedAccountBlockBySbHeight(head.Height+1, &self.address)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return block
}

type snapshotCh struct {
	bc      ch.Chain
	version *ForkVersion
}

func (self *snapshotCh) getBlock(height uint64) commonBlock {
	block, e := self.bc.GetSnapshotBlockByHeight(height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, self.version)
}

func (self *snapshotCh) head() commonBlock {
	block := self.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, self.version)
}

func (self *snapshotCh) headSnapshot() *ledger.SnapshotBlock {
	block := self.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) getSnapshotBlockByHash(hash types.Hash) *ledger.SnapshotBlock {
	block, e := self.bc.GetSnapshotBlockByHash(&hash)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	ss, bm, e := self.bc.DeleteSnapshotBlocksToHeight(height)
	if e != nil {
		return nil, nil, e
	}

	var accountResults map[types.Address][]commonBlock
	for addr, bs := range bm {
		var r []commonBlock
		for _, b := range bs {
			r = append(r, newAccountPoolBlock(b, nil, self.version))
		}
		accountResults[addr] = r
	}
	var snapshotResults []commonBlock
	for _, s := range ss {
		snapshotResults = append(snapshotResults, newSnapshotPoolBlock(s, self.version))
	}
	return snapshotResults, accountResults, nil
}

func (self *snapshotCh) insertBlock(block commonBlock) error {
	b := block.(*snapshotPoolBlock)
	return self.bc.InsertSnapshotBlock(b.block)
}

func (self *snapshotCh) insertBlocks(bs []commonBlock) error {
	monitor.LogEvent("pool", "NonSnapshot")
	for _, b := range bs {
		err := self.insertBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}
