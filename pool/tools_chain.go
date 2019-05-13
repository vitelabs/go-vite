package pool

import (
	"fmt"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/common"

	ch "github.com/vitelabs/go-vite/chain"
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
	DeleteAccountBlocksToHeight(addr types.Address, toHeight uint64) ([]*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	IsGenesisSnapshotBlock(hash types.Hash) bool
	IsGenesisAccountBlock(block types.Hash) bool
	GetQuotaUnused(address types.Address) (uint64, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
	GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error)
	SetConsensus(cs ch.Consensus)
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
	version *common.Version
	log     log15.Logger
}

func (accCh *accountCh) insertBlock(b commonBlock) error {
	monitor.LogEvent("pool", "insertChain")
	block := b.(*accountPoolBlock)
	accountBlock := &vm_db.VmAccountBlock{AccountBlock: block.block, VmDb: block.vmBlock}
	return accCh.rw.InsertAccountBlock(accountBlock)
}

func (accCh *accountCh) head() commonBlock {
	block, e := accCh.rw.GetLatestAccountBlock(accCh.address)
	if e != nil {
		accCh.log.Error("get latest block error", "error", e)
		return nil
	}
	if block == nil {
		return nil
	}

	result := newAccountPoolBlock(block, nil, accCh.version, types.RollbackChain)
	return result
}

func (accCh *accountCh) getBlock(height uint64) commonBlock {
	if height == types.EmptyHeight {
		return newAccountPoolBlock(&ledger.AccountBlock{Height: types.EmptyHeight}, nil, accCh.version, types.RollbackChain)
	}
	defer monitor.LogTime("pool", "getAccountBlock", time.Now())
	// todo
	block, e := accCh.rw.GetAccountBlockByHeight(accCh.address, height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}

	return newAccountPoolBlock(block, nil, accCh.version, types.RollbackChain)
}

func (accCh *accountCh) insertBlocks(bs []commonBlock) error {
	var blocks []*vm_db.VmAccountBlock
	for _, b := range bs {
		block := b.(*accountPoolBlock)
		accCh.log.Info(fmt.Sprintf("account block insert. [%s][%d][%s].\n", block.block.AccountAddress, block.Height(), block.Hash()))
		monitor.LogEvent("pool", "insertChain")
		blocks = append(blocks, &vm_db.VmAccountBlock{AccountBlock: block.block, VmDb: block.vmBlock})
		monitor.LogEvent("pool", "accountInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	}

	for _, v := range blocks {
		err := accCh.rw.InsertAccountBlock(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (accCh *accountCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	accCh.log.Info("delToHeight", "height", height, "address", accCh.address)
	bm, e := accCh.rw.DeleteAccountBlocksToHeight(accCh.address, height)
	if e != nil {
		return nil, nil, e
	}

	// FIXME
	results := make(map[types.Address][]commonBlock)
	for _, b := range bm {
		results[b.AccountAddress] = append(results[b.AccountAddress], newAccountPoolBlock(b, nil, accCh.version, types.RollbackChain))
		accCh.log.Info("actual delToHeight", "height", b.Height, "hash", b.Hash, "address", b.AccountAddress)
	}
	block, err := accCh.rw.GetLatestAccountBlock(accCh.address)
	if err != nil {
		panic(err)
	}
	if block.Height > height {
		panic(fmt.Sprintf("delete fail.%d-%d", block.Height, height))
	}
	return nil, results, nil
}

func (accCh *accountCh) getUnConfirmedBlocks() []*ledger.AccountBlock {
	return accCh.rw.GetUnconfirmedBlocks(accCh.address)
}

func (accCh *accountCh) getLatestSnapshotBlock() *ledger.SnapshotBlock {
	return accCh.rw.GetLatestSnapshotBlock()
}

func (accCh *accountCh) getQuotaUnused() uint64 {
	// todo
	unused, err := accCh.rw.GetQuotaUnused(accCh.address)
	if err != nil {
		accCh.log.Error("get account quota err", "err", err)
		return 0
	}
	return unused
}
func (accCh *accountCh) getConfirmedTimes(abHash types.Hash) (uint64, error) {
	return accCh.rw.GetConfirmedTimes(abHash)
}
func (accCh *accountCh) needSnapshot(addr types.Address) (uint8, error) {
	meta, err := accCh.rw.GetContractMeta(addr)
	if err != nil {
		return 0, err
	}
	if meta == nil {
		accCh.log.Warn("meta info is nil.", "addr", addr)
		return 0, nil
	}
	return meta.SendConfirmedTimes, nil
}

type snapshotCh struct {
	bc      chainDb
	version *common.Version
	log     log15.Logger
}

func (sCh *snapshotCh) getBlock(height uint64) commonBlock {
	defer monitor.LogTime("pool", "getSnapshotBlock", time.Now())
	block, e := sCh.bc.GetSnapshotHeaderByHeight(height)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, sCh.version, types.QueryChain)
}

func (sCh *snapshotCh) head() commonBlock {
	block := sCh.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, sCh.version, types.RollbackChain)
}

func (sCh *snapshotCh) headSnapshot() *ledger.SnapshotBlock {
	block := sCh.bc.GetLatestSnapshotBlock()
	if block == nil {
		return nil
	}
	return block
}

func (sCh *snapshotCh) getSnapshotBlockByHash(hash types.Hash) *ledger.SnapshotBlock {
	block, e := sCh.bc.GetSnapshotBlockByHash(hash)
	if e != nil {
		return nil
	}
	if block == nil {
		return nil
	}
	return block
}

func (sCh *snapshotCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	schunk, e := sCh.bc.DeleteSnapshotBlocksToHeight(height)

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
			accountResults[b.AccountAddress] = append(blocks, newAccountPoolBlock(b, nil, sCh.version, types.RollbackChain))
		}
	}
	var snapshotResults []commonBlock
	for _, s := range schunk {
		if s.SnapshotBlock == nil {
			continue
		}
		snapshotResults = append(snapshotResults, newSnapshotPoolBlock(s.SnapshotBlock, sCh.version, types.RollbackChain))
	}
	return snapshotResults, accountResults, nil
}

func (sCh *snapshotCh) insertBlock(block commonBlock) error {
	panic("not implement")
}

func (sCh *snapshotCh) insertSnapshotBlock(b *snapshotPoolBlock) (map[types.Address][]commonBlock, error) {
	if b.Source() == types.QueryChain {
		sCh.log.Crit("QueryChain insert to chain.", "Height", b.Height(), "Hash", b.Hash())
	}
	monitor.LogEvent("pool", "insertChain")
	monitor.LogEvent("pool", "snapshotInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	bm, err := sCh.bc.InsertSnapshotBlock(b.block)

	results := make(map[types.Address][]commonBlock)
	for _, bs := range bm {

		result, ok := results[bs.AccountAddress]
		if !ok {
			var r []commonBlock
			result = r
		}
		results[bs.AccountAddress] = append(result, newAccountPoolBlock(bs, nil, sCh.version, types.RollbackChain))
		sCh.log.Info("account block delete by insertToSnapshot.",
			"snapshot", fmt.Sprintf("[%d][%s]", b.Height(), b.Hash()),
			"aBlock", fmt.Sprintf("[%s][%d][%s]", bs.AccountAddress, bs.Height, bs.Hash))
	}
	return results, err
}

func (sCh *snapshotCh) insertBlocks(bs []commonBlock) error {
	monitor.LogEvent("pool", "NonSnapshot")
	for _, b := range bs {
		monitor.LogEvent("pool", "snapshotInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
		err := sCh.insertBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}
