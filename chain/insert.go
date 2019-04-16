package chain

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (c *chain) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	//FOR DEBUG
	//c.log.Info(fmt.Sprintf("insert ab %s %d %s %s\n", vmAccountBlock.AccountBlock.AccountAddress, vmAccountBlock.AccountBlock.Height, vmAccountBlock.AccountBlock.Hash, vmAccountBlock.AccountBlock.FromBlockHash))

	vmAbList := []*vm_db.VmAccountBlock{vmAccountBlock}
	if err := c.em.Trigger(prepareInsertAbsEvent, vmAbList, nil, nil, nil); err != nil {
		return err
	}

	accountBlock := vmAccountBlock.AccountBlock
	// write cache
	c.cache.InsertAccountBlock(accountBlock)

	// write index database
	if err := c.indexDB.InsertAccountBlock(accountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlockAndSnapshot failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlockAndSnapshot")
	}

	// write state db
	if err := c.stateDB.Write(vmAccountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.WriteAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlockAndSnapshot")
	}

	c.em.Trigger(insertAbsEvent, vmAbList, nil, nil, nil)

	return nil
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error) {
	//FOR DEBUG
	//c.log.Info(fmt.Sprintf("Insert snapshot block %d %s\n", snapshotBlock.Height, snapshotBlock.Hash))

	//for Addr, hh := range snapshotBlock.SnapshotContent {
	//	c.log.Info(fmt.Sprintf("Insert %d SC: %s %d %s\n", snapshotBlock.Height, Addr, hh.Height, hh.Hash))
	//}

	canBeSnappedBlocks, err := c.getBlocksToBeConfirmed(snapshotBlock.SnapshotContent)
	if err != nil {
		return nil, err
	}

	sbList := []*ledger.SnapshotBlock{snapshotBlock}

	if err := c.em.Trigger(prepareInsertSbsEvent, nil, nil, sbList, nil); err != nil {
		return nil, err
	}

	// write block db
	abLocationList, snapshotBlockLocation, err := c.blockDB.Write(&ledger.SnapshotChunk{
		SnapshotBlock: snapshotBlock,
		AccountBlocks: canBeSnappedBlocks,
	})

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.WriteAccountBlock failed, snapshotBlock is %+v. Error: %s", snapshotBlock, err.Error()))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	// insert index
	c.indexDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, snapshotBlockLocation, abLocationList)

	// update latest snapshot block cache
	c.cache.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks)

	// insert snapshot blocks
	c.stateDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks)

	c.em.Trigger(InsertSbsEvent, nil, nil, sbList, nil)

	// delete invalidBlocks
	invalidBlocks := c.filterUnconfirmedBlocks(true)

	if len(invalidBlocks) > 0 {
		if err := c.deleteAccountBlocks(invalidBlocks); err != nil {
			return nil, err
		}
	}

	c.flusher.Flush(false)

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

	return blocks, errors.New(fmt.Sprintf("lack block, sc is %s", sPrintError(sc, blocks)))
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
