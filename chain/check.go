package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain/state"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func printSnapshotLog(snapshotLog chain_state.SnapshotLog) string {
	str := ""
	for addr, logItems := range snapshotLog {
		str += fmt.Sprintf("%s: %d, ", addr, len(logItems))
	}
	return str
}

func (c *chain) CheckRedo() error {
	redoStore := c.stateDB.RedoStore()
	iter := redoStore.NewIterator(nil)

	redo := c.stateDB.StorageRedo()

	var prevHeight uint64

	for iter.Next() {
		key := iter.Key()
		snapshotHeight := chain_utils.BytesToUint64(key[1:])

		if prevHeight > 0 && prevHeight+1 != snapshotHeight {
			return errors.New(fmt.Sprintf("prevHeight + 1 != snapshotHeight, prev height is %d, snapshot height is %d", prevHeight, snapshotHeight))
		}
		prevHeight = snapshotHeight

		chunks, err := c.GetSubLedger(snapshotHeight-1, snapshotHeight)
		if err != nil {
			return errors.New(fmt.Sprintf("c.GetSubLedger failed, start snapshot height is %d, end snapshot height is %d", snapshotHeight-1, snapshotHeight))
		}
		snapshotLog, ok, err := redo.QueryLog(snapshotHeight)
		if err != nil {
			return errors.New(fmt.Sprintf("redo.QueryLog failed, snapshot height is %d", snapshotHeight))
		}
		if !ok {
			return errors.New(fmt.Sprintf("ok is false, snapshot height is %d", snapshotHeight))
		}

		blockCount := 0
		for _, chunk := range chunks {
			blockCount += len(chunk.AccountBlocks)
			for _, accountBlock := range chunk.AccountBlocks {
				if logs, ok := snapshotLog[accountBlock.AccountAddress]; !ok || len(logs) <= 0 {
					return errors.New(fmt.Sprintf("!ok || len(logs) <= 0. snapshot log is %s. accountBlock is %+v, snapshot height is %d",
						printSnapshotLog(snapshotLog), accountBlock, snapshotHeight))
				}
			}
		}

		logLength := 0
		for _, logItems := range snapshotLog {
			logLength += len(logItems)
		}

		if blockCount != logLength {
			return errors.New(fmt.Sprintf("blockCount != logLength, blockCount is %d, logLength is %d, snapshot log is %s", blockCount, logLength, printSnapshotLog(snapshotLog)))
		}

		c.log.Info(fmt.Sprintf("snapshot height: %d. %d logs, %d blocks", snapshotHeight, logLength, blockCount), "method", "checkRedo")
	}
	err := iter.Error()
	iter.Release()
	return err
}

func (c *chain) CheckRecentBlocks() error {
	latestSb := c.GetLatestSnapshotBlock()
	c.log.Info(fmt.Sprintf("latest snapshot block is %d, %s", latestSb.Height, latestSb.Hash), "method", "checkRecentBlocks")
	sbList, err := c.GetSnapshotBlocks(latestSb.Hash, false, 100)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetSnapshotBlocks failed. Error: %s", err))
		return cErr
	}

	var prevSb *ledger.SnapshotBlock

	accountLatestBlockMap := make(map[types.Address]*ledger.AccountBlock)
	for _, sb := range sbList {
		if prevSb != nil && (prevSb.PrevHash != sb.Hash || prevSb.Height != sb.Height+1) {
			return errors.New(fmt.Sprintf("prevSb is %+v, sb is %+v", prevSb, sb))
		}

		prevSb = sb

		c.log.Info(fmt.Sprintf("check snapshot block %d, %s", sb.Height, sb.Hash), "method", "checkRecentBlocks")

		for addr, hashHeight := range sb.SnapshotContent {

			block, err := c.GetAccountBlockByHash(hashHeight.Hash)
			if err != nil {
				return errors.New(fmt.Sprintf("c.GetAccountBlockByHash failed, addr is %s, hash is %s. Error: %s", addr, hashHeight.Hash, err))
			}

			for {
				if block == nil {
					return errors.New(fmt.Sprintf("c.GetAccountBlockByHash(), block is nil, addr is %s, hash is %s", addr, hashHeight.Hash))
				}

				c.log.Info(fmt.Sprintf("check account block %s %d %s %s", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash))

				cacheLatestBlock, ok := accountLatestBlockMap[addr]
				if !ok {
					latestBlock, err := c.GetLatestAccountBlock(addr)
					if err != nil {
						return errors.New(fmt.Sprintf("c.GetLatestAccountBlock failed, addr is %s. Error: %s", addr, err))
					}

					if latestBlock == nil {
						return errors.New(fmt.Sprintf("c.GetAccountBlockByHash(), latest account block is nil, addr is %s", addr))
					}

					if latestBlock.Hash != block.Hash {
						return errors.New(fmt.Sprintf("latest account block is %+v, block is %+v", latestBlock, block))
					}

				} else if cacheLatestBlock.Height <= block.Height {
					return errors.New(fmt.Sprintf("cacheLatestBlock.Height <= block.Height, cacheLatestBlock is %+v, block is %+v", cacheLatestBlock, block))
				}
				// set latest
				accountLatestBlockMap[addr] = block

				// get prev block
				if block.Height <= 1 {
					break
				}

				confirmedSb, err := c.GetConfirmSnapshotBlockByAbHash(block.PrevHash)
				if err != nil {
					return errors.New(fmt.Sprintf("GetConfirmSnapshotBlockByAbHash failed, addr is %s, prevHash is %s. Error: %s", addr, block.PrevHash, err))
				}

				if confirmedSb == nil {
					return errors.New(fmt.Sprintf("confirmd sb is nil, account block hash is %s", block.PrevHash))
				}
				if confirmedSb.Hash != sb.Hash {
					break
				}

				prevBlock, err := c.GetAccountBlockByHash(block.PrevHash)
				if err != nil {
					return errors.New(fmt.Sprintf("get prev account block failed, addr is %s, prevHash is %s. Error: %s", addr, block.PrevHash, err))
				}
				if prevBlock == nil {
					return errors.New(fmt.Sprintf("get prev account block is nil, addr is %s, prevHash is %s. Error: %s", addr, block.PrevHash, err))
				}

				block = prevBlock

			}
		}

	}

	return nil
}
