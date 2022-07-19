package chain

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (c *chain) InsertAccountBlock(vmAccountBlock *interfaces.VmAccountBlock) error {
	c.flushMu.RLock()
	defer c.flushMu.RUnlock()

	// FOR DEBUG
	c.log.Info(fmt.Sprintf("insert account block %s %d %s %s\n", vmAccountBlock.AccountBlock.AccountAddress, vmAccountBlock.AccountBlock.Height, vmAccountBlock.AccountBlock.Hash, vmAccountBlock.AccountBlock.FromBlockHash))

	vmAbList := []*interfaces.VmAccountBlock{vmAccountBlock}
	if err := c.em.TriggerInsertAbs(prepareInsertAbsEvent, vmAbList); err != nil {
		return err
	}

	accountBlock := vmAccountBlock.AccountBlock
	// write cache
	c.cache.InsertAccountBlock(accountBlock)

	// write index database
	if err := c.indexDB.InsertAccountBlock(accountBlock); err != nil {
		cErr := fmt.Errorf("c.indexDB.InsertAccountBlockAndSnapshot failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash)
		common.Crit(cErr.Error(), "method", "InsertAccountBlockAndSnapshot")
	}

	// write state db
	if err := c.stateDB.Write(vmAccountBlock); err != nil {
		cErr := fmt.Errorf("c.stateDB.WriteAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash)
		common.Crit(cErr.Error(), "method", "InsertAccountBlockAndSnapshot")
	}

	if err := c.em.TriggerInsertAbs(insertAbsEvent, vmAbList); err != nil {
		cErr := fmt.Errorf("trigger InsertAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash)
		c.log.Error(cErr.Error())
	}
	return nil
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error) {
	// FOR DEBUG
	c.log.Info(fmt.Sprintf("insert snapshot block %s %d\n", snapshotBlock.Hash, snapshotBlock.Height))
	if err := c.insertSnapshotBlock(snapshotBlock); err != nil {
		return nil, err
	}

	// delete invalidBlocks
	invalidBlocks := c.filterUnconfirmedBlocks(snapshotBlock, true)

	if len(invalidBlocks) > 0 {
		if err := c.deleteAccountBlocks(invalidBlocks); err != nil {
			return nil, err
		}
	}

	c.cache.ResetUnconfirmedQuotas(c.GetAllUnconfirmedBlocks())

	// only trigger
	return invalidBlocks, nil
}

func (c *chain) getBlocksToBeConfirmed(sc ledger.SnapshotContent) ([]*ledger.AccountBlock, error) {
	scLen := len(sc)
	if scLen <= 0 {
		return nil, nil
	}

	blocks := c.cache.GetUnconfirmedBlocks()
	finishCount := 0
	blocksToBeConfirmed := make([]*ledger.AccountBlock, 0, len(blocks))
	for _, block := range blocks {
		if hashHeight, ok := sc[block.AccountAddress]; ok {
			if block.Height < hashHeight.Height {
				blocksToBeConfirmed = append(blocksToBeConfirmed, block)
			} else if block.Height == hashHeight.Height {
				blocksToBeConfirmed = append(blocksToBeConfirmed, block)
				finishCount += 1
			}
		}
		if finishCount >= scLen {
			return blocksToBeConfirmed, nil
		}
	}

	return blocks, fmt.Errorf("lack block, sc is %s", sPrintError(sc, blocks))
}

func (c *chain) insertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
	c.flushMu.RLock()
	defer func() {
		if e := recover(); e != nil {
			c.flusher.Abort()
			c.flushMu.RUnlock()
			panic(e)
		}

		c.flushMu.RUnlock()
	}()

	canBeSnappedBlocks, err := c.getBlocksToBeConfirmed(snapshotBlock.SnapshotContent)
	if err != nil {
		return err
	}

	//sbList := []*ledger.SnapshotBlock{snapshotBlock}
	chunks := []*ledger.SnapshotChunk{{
		SnapshotBlock: snapshotBlock,
		AccountBlocks: canBeSnappedBlocks,
	}}

	if err := c.em.TriggerInsertSbs(prepareInsertSbsEvent, chunks); err != nil {
		return err
	}

	// write block db
	abLocationMap, snapshotBlockLocation, err := c.blockDB.Write(chunks[0])

	if err != nil {
		cErr := fmt.Errorf("c.blockDB.WriteAccountBlock failed, snapshotBlock is %+v. Error: %s", snapshotBlock, err.Error())
		common.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// insert index
		c.indexDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, snapshotBlockLocation, abLocationMap)

		// update latest snapshot block cache
		c.cache.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks)
	}()

	go func() {
		defer wg.Done()
		// insert snapshot blocks
		c.stateDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks)
	}()

	wg.Wait()

	c.em.TriggerInsertSbs(InsertSbsEvent, chunks)
	return nil
}

func sPrintError(sc ledger.SnapshotContent, blocks []*ledger.AccountBlock) string {
	str := "SnapshotContent: "
	for addr, hashHeight := range sc {
		str += addr.String() + " " + hashHeight.Hash.String() + " " + strconv.FormatUint(hashHeight.Height, 10) + ", "
	}

	str += "| Blocks: "
	for _, block := range blocks {
		str += fmt.Sprintf("%+v, ", block)
	}
	return str
}
