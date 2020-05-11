package pool

import (
	"fmt"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/common/helper"

	"github.com/vitelabs/go-vite/interval/utils"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
)

type chainDb interface {
	face.AccountReader
	face.AccountWriter
	face.SnapshotWriter
	face.SnapshotReader
}

type chainRw interface {
	insertBlock(block commonBlock) error
	insertBlocks(blocks []commonBlock) error

	head() commonBlock
	getBlock(height uint64) commonBlock
	getHash(height uint64) *common.Hash
}

type accountCh struct {
	address common.Address
	rw      chainDb
	version *common.Version
	log     log15.Logger
}

func (accCh *accountCh) insertBlock(b commonBlock) error {
	monitor.LogEvent("pool", "insertChain")
	block := b.(*accountPoolBlock)
	sendInfo := "none"
	reqInfo := "none"
	if block.block.Source != nil {
		reqInfo = block.block.Source.Hash.String()
	}
	accCh.log.Info("insert account block", "addr", block.block.Signer(), "height", block.block.Height, "hash", block.block.Hash, "sendList", sendInfo, "requestHash", reqInfo)
	return accCh.rw.InsertAccountBlock(accCh.address, block.block)
}

func (accCh *accountCh) head() commonBlock {
	block, e := accCh.rw.HeadAccount(accCh.address)
	if e != nil {
		accCh.log.Error("get latest block error", "error", e)
		return nil
	}
	if block == nil {
		return nil
	}

	result := newAccountPoolBlock(block, accCh.version, types.RollbackChain)
	return result
}

func (accCh *accountCh) getBlock(height uint64) commonBlock {
	if height == types.EmptyHeight {
		return newAccountPoolBlock(common.NewAccountBlockEmpty(), accCh.version, types.RollbackChain)
	}
	defer monitor.LogTime("pool", "getAccountBlock", time.Now())
	// todo
	block := accCh.rw.GetAccountByHeight(accCh.address, common.Height(height))
	if block == nil {
		return nil
	}

	return newAccountPoolBlock(block, accCh.version, types.RollbackChain)
}

func (accCh *accountCh) getHash(height uint64) *common.Hash {
	if height == types.EmptyHeight {
		return &common.EmptyHash
	}
	block := accCh.getBlock(height)
	if block != nil {
		hash := block.Hash()
		return &hash
	}
	return &common.EmptyHash
}

func (accCh *accountCh) insertBlocks(bs []commonBlock) error {
	var blocks []*common.AccountStateBlock
	for _, b := range bs {
		block := b.(*accountPoolBlock)
		accCh.log.Info(fmt.Sprintf("account block insert. [%s][%d][%s].\n", block.block.Signer().String(), block.Height(), block.Hash()))
		monitor.LogEvent("pool", "insertChain")
		blocks = append(blocks, block.block)
		monitor.LogEvent("pool", "accountInsertSource_"+strconv.FormatUint(uint64(b.Source()), 10))
	}

	for _, v := range blocks {
		err := accCh.rw.InsertAccountBlock(accCh.address, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (accCh *accountCh) delToHeight(height uint64) ([]commonBlock, map[common.Address][]commonBlock, error) {
	accCh.log.Info("delToHeight", "height", height, "address", accCh.address)
	bm, e := accCh.rw.RollbackAccountBlockTo(accCh.address, common.Height(height))
	if e != nil {
		return nil, nil, e
	}

	// FIXME
	results := make(map[common.Address][]commonBlock)
	for _, b := range bm {
		results[b.Signer()] = append(results[b.Signer()], newAccountPoolBlock(b, accCh.version, types.RollbackChain))
		accCh.log.Info("actual delToHeight", "height", b.Height, "hash", b.Hash, "address", b.Signer())
	}

	{ // todo delete
		block, err := accCh.rw.HeadAccount(accCh.address)
		if err != nil {
			panic(err)
		}
		if height == 1 {
			if block != nil {
				panic(fmt.Sprintf("latest block should be nil"))
			}
		} else {
			if block == nil {
				panic(fmt.Sprintf("latest block is nil"))
			}
			if block.Height().Uint64() > height {
				panic(fmt.Sprintf("delete fail.%d-%d", block.Height, height))
			}
		}
	}
	return nil, results, nil
}

func (accCh *accountCh) getLatestSnapshotBlock() *common.SnapshotBlock {
	snapshot, err := accCh.rw.HeadSnapshot()
	utils.Nil(err)

	return snapshot
}

func (accCh *accountCh) getQuotaUnused() uint64 {
	return helper.MaxUint64
}
func (accCh *accountCh) getConfirmedTimes(abHash common.Hash) (uint64, error) {
	return 0, nil
}
func (accCh *accountCh) needSnapshot(addr common.Address) (uint8, error) {
	return 0, nil
}

type snapshotCh struct {
	bc      chainDb
	version *common.Version
	log     log15.Logger
}

func (sCh *snapshotCh) getBlock(height uint64) commonBlock {
	block := sCh.bc.GetSnapshotByHeight(common.Height(height))
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, sCh.version, types.QueryChain)
}

func (sCh *snapshotCh) getHash(height uint64) *common.Hash {
	block := sCh.getBlock(height)
	if block == nil {
		return nil
	}
	hash := block.Hash()
	return &hash
}

func (sCh *snapshotCh) head() commonBlock {
	block, err := sCh.bc.HeadSnapshot()
	utils.Nil(err)
	if block == nil {
		return nil
	}
	return newSnapshotPoolBlock(block, sCh.version, types.RollbackChain)
}

func (sCh *snapshotCh) headSnapshot() *common.SnapshotBlock {
	block, err := sCh.bc.HeadSnapshot()
	utils.Nil(err)
	if block == nil {
		return nil
	}
	return block
}

func (sCh *snapshotCh) getSnapshotBlockByHash(hash common.Hash) *common.SnapshotBlock {
	block := sCh.bc.GetSnapshotByHash(hash)
	return block
}

func (sCh *snapshotCh) delToHeight(height uint64) ([]commonBlock, map[common.Address][]commonBlock, error) {
	snaps, acctsMap, e := sCh.bc.RollbackSnapshotBlockTo(common.Height(height))

	if e != nil {
		return nil, nil, e
	}

	accountResults := make(map[common.Address][]commonBlock)
	for addr, bs := range acctsMap {
		for _, b := range bs {
			accountResults[addr] = append(accountResults[addr], newAccountPoolBlock(b, sCh.version, types.RollbackChain))
		}
	}
	var snapshotResults []commonBlock
	for _, s := range snaps {
		snapshotResults = append(snapshotResults, newSnapshotPoolBlock(s, sCh.version, types.RollbackChain))
	}
	return snapshotResults, accountResults, nil
}

func (sCh *snapshotCh) insertBlock(block commonBlock) error {
	panic("not implement")
}

func (sCh *snapshotCh) insertSnapshotBlock(b *snapshotPoolBlock) (map[common.Address][]commonBlock, error) {
	if b.Source() == types.QueryChain {
		sCh.log.Crit("QueryChain insert to chain.", "Height", b.Height(), "Hash", b.Hash())
	}
	err := sCh.bc.InsertSnapshotBlock(b.block)

	results := make(map[common.Address][]commonBlock)
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
