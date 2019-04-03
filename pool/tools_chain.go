package pool

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_db"
)

type chainDb interface {
	InsertAccountBlock(vmAccountBlocks *vm_db.VmAccountBlock) error
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr types.Address, height uint64) (*ledger.AccountBlock, error)
	//DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error)
	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock
	//GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error)
	//GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotHeaderByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotHeaderByHash(hash types.Hash) (*ledger.SnapshotBlock, error)
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error)
	DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	IsGenesisSnapshotBlock(hash types.Hash) bool
	IsGenesisAccountBlock(block types.Hash) bool
	GetQuotaUnused(address types.Address) (uint64, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)
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
	log     log15.Logger
}

func (self *accountCh) insertBlock(b commonBlock) error {
	if !b.checkForkVersion() {
		return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.FormatInt(int64(b.forkVersion()), 10))
	}

	monitor.LogEvent("pool", "accountInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	block := b.(*accountPoolBlock)
	accountBlock := &vm_db.VmAccountBlock{AccountBlock: block.block, VmDb: block.vmBlock}
	return self.rw.InsertAccountBlock(accountBlock)
}

func (self *accountCh) head() commonBlock {
	block, e := self.rw.GetLatestAccountBlock(self.address)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}

	result := newAccountPoolBlock(block, nil, self.version, types.RollbackChain)
	return result
}

func (self *accountCh) getBlock(height uint64) commonBlock {
	if height == types.EmptyHeight {
		return newAccountPoolBlock(&ledger.AccountBlock{Height: types.EmptyHeight}, nil, self.version, types.RollbackChain)
	}
	defer monitor.LogTime("pool", "getAccountBlock", time.Now())
	// todo
	block, e := self.rw.GetAccountBlockByHeight(self.address, height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}

	return newAccountPoolBlock(block, nil, self.version, types.RollbackChain)
}

func (self *accountCh) insertBlocks(bs []commonBlock) error {
	var blocks []*vm_db.VmAccountBlock
	for _, b := range bs {
		block := b.(*accountPoolBlock)
		fmt.Printf("account block insert. [%s][%d][%s].\n", block.block.AccountAddress, block.Height(), block.Hash())
		blocks = append(blocks, &vm_db.VmAccountBlock{AccountBlock: block.block, VmDb: block.vmBlock})
		monitor.LogEvent("pool", "accountInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	}

	for _, v := range blocks {
		err := self.rw.InsertAccountBlock(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *accountCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	// todo
	return nil, nil, nil
	//bm, e := self.rw.DeleteAccountBlocks(&self.address, height)
	//if e != nil {
	//	return nil, nil, e
	//}
	//
	//// FIXME
	//results := make(map[types.Address][]commonBlock)
	//for addr, bs := range bm {
	//	var r []commonBlock
	//	for _, b := range bs {
	//		r = append(r, newAccountPoolBlock(b, nil, self.version, types.RollbackChain))
	//	}
	//	results[addr] = r
	//}
	//return nil, results, nil
}

func (self *accountCh) getUnConfirmedBlocks() []*ledger.AccountBlock {
	return self.rw.GetUnconfirmedBlocks(self.address)
}

func (self *accountCh) getLatestSnapshotBlock() *ledger.SnapshotBlock {
	return self.rw.GetLatestSnapshotBlock()
}

func (self *accountCh) getFirstUnconfirmedBlock(head *ledger.SnapshotBlock) *ledger.AccountBlock {
	//block := self.rw.GetUnconfirmedBlocks(self.address)
	//if block == nil {
	//	return nil
	//}
	//return block
	// todo
	return nil
}
func (self *accountCh) getQuotaUnused() uint64 {
	// todo
	unused, err := self.rw.GetQuotaUnused(self.address)
	if err != nil {
		self.log.Error("get account quota err", "err", err)
		return 0
	}
	return unused
}
func (self *accountCh) getConfirmedTimes(abHash types.Hash) (uint64, error) {
	return self.rw.GetConfirmedTimes(abHash)
}
func (self *accountCh) needSnapshot(addr types.Address) (uint8, error) {
	meta, err := self.rw.GetContractMeta(addr)
	if err != nil {
		return 0, err
	}
	return meta.SendConfirmedTimes, nil
}

type snapshotCh struct {
	bc      chainDb
	version *ForkVersion
	log     log15.Logger
}

func (self *snapshotCh) getBlock(height uint64) commonBlock {
	defer monitor.LogTime("pool", "getSnapshotBlock", time.Now())
	block, e := self.bc.GetSnapshotHeaderByHeight(height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, self.version, types.QueryChain)
}

func (self *snapshotCh) head() commonBlock {
	block := self.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, self.version, types.RollbackChain)
}

func (self *snapshotCh) headSnapshot() *ledger.SnapshotBlock {
	block := self.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) getSnapshotBlockByHash(hash types.Hash) *ledger.SnapshotBlock {
	block, e := self.bc.GetSnapshotBlockByHash(hash)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	schunk, e := self.bc.DeleteSnapshotBlocksToHeight(height)

	if e != nil {
		return nil, nil, e
	}

	accountResults := make(map[types.Address][]commonBlock)
	for _, bs := range schunk {
		for _, b := range bs.AccountBlocks {
			blocks, ok := accountResults[b.AccountAddress]
			if !ok {
				var r []commonBlock
				blocks = r
			}
			accountResults[b.AccountAddress] = append(blocks, newAccountPoolBlock(b, nil, self.version, types.RollbackChain))
		}
	}
	var snapshotResults []commonBlock
	for _, s := range schunk {
		snapshotResults = append(snapshotResults, newSnapshotPoolBlock(s.SnapshotBlock, self.version, types.RollbackChain))
	}
	return snapshotResults, accountResults, nil
}

func (self *snapshotCh) insertBlock(block commonBlock) error {
	//b := block.(*snapshotPoolBlock)
	//if b.Source() == types.QueryChain {
	//	self.log.Crit("QueryChain insert to chain.", "Height", b.Height(), "Hash", b.Hash())
	//}
	//monitor.LogEvent("pool", "snapshotInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	//return self.bc.InsertSnapshotBlock(b.block)
	panic("not implement")
}

func (self *snapshotCh) insertSnapshotBlock(b *snapshotPoolBlock) (map[types.Address][]commonBlock, error) {
	if b.Source() == types.QueryChain {
		self.log.Crit("QueryChain insert to chain.", "Height", b.Height(), "Hash", b.Hash())
	}
	monitor.LogEvent("pool", "snapshotInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	bm, err := self.bc.InsertSnapshotBlock(b.block)

	results := make(map[types.Address][]commonBlock)
	for _, bs := range bm {

		result, ok := results[bs.AccountAddress]
		if !ok {
			var r []commonBlock
			result = r
		}
		results[bs.AccountAddress] = append(result, newAccountPoolBlock(bs, nil, self.version, types.RollbackChain))
	}
	return results, err
}

func (self *snapshotCh) insertBlocks(bs []commonBlock) error {
	monitor.LogEvent("pool", "NonSnapshot")
	for _, b := range bs {
		monitor.LogEvent("pool", "snapshotInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
		err := self.insertBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}
