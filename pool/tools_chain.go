package pool

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
)

type chainDb interface {
	InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)
	DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)
	GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock
	GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
}

type chainRw interface {
	insertBlock(block commonBlock) error
	insertBlocks(blocks []commonBlock) error

	head() commonBlock
	getBlock(height uint64) commonBlock
}

type accountCh struct {
	address types.Address
	rw      chainDb
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
	if height == types.EmptyHeight {
		return newAccountPoolBlock(&ledger.AccountBlock{Height: types.EmptyHeight}, nil, self.version)
	}
	// todo
	block, e := self.rw.GetAccountBlockByHeight(&self.address, height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}

	return newAccountPoolBlock(block, nil, self.version)
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

	// FIXME
	results := make(map[types.Address][]commonBlock)
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
	bc      chainDb
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

	accountResults := make(map[types.Address][]commonBlock)
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
