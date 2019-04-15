package chain

import (
	"encoding/binary"
	"fmt"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
	"time"
)

func TestChain_SnapshotBlock(t *testing.T) {
	chainInstance, accounts, snapshotBlockList := SetUp(t, 10, 1234, 7)

	testSnapshotBlock(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func IsSnapshotBlockExisted(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		result, err := chainInstance.IsSnapshotBlockExisted(snapshotBlock.Hash)
		if err != nil {
			panic(err)
		}
		if !result {
			panic("error")
		}
	}

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))
		if err != nil {
			panic(err)
		}
		result, err := chainInstance.IsSnapshotBlockExisted(hash)
		if err != nil {
			panic(err)
		}
		if result {
			panic("error")
		}
	}

}

func IsGenesisSnapshotBlock(chainInstance *chain) {
	genesisSnapshotBlock := chainInstance.GetGenesisSnapshotBlock()
	if !chainInstance.IsGenesisSnapshotBlock(genesisSnapshotBlock.Hash) {
		panic("error")
	}

	for i := 0; i < 100; i++ {
		hash, err := types.BytesToHash(crypto.Hash256(chain_utils.Uint64ToBytes(uint64(time.Now().UnixNano()))))
		if err != nil {
			if chainInstance.IsGenesisSnapshotBlock(hash) {
				panic("error")
			}
		}
	}
}

func GetGenesisSnapshotBlock(chainInstance *chain) {
	genesisSnapshotBlock := chainInstance.GetGenesisSnapshotBlock()

	correctFirstSb, err := chainInstance.GetSnapshotBlockByHeight(1)
	if err != nil {
		panic(err)
	}

	if genesisSnapshotBlock.Hash != correctFirstSb.Hash {
		chainInstance.GetSnapshotBlockByHeight(1)
		panic(fmt.Sprintf("%+v. %+v\n", genesisSnapshotBlock, correctFirstSb))
	}
}

func GetLatestSnapshotBlock(chainInstance *chain) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	nextSb, err := chainInstance.GetSnapshotHeaderByHeight(latestSnapshotBlock.Height + 1)
	if err != nil {
		panic(err)
	}
	if nextSb != nil {
		panic("error")
	}
}

func GetSnapshotHeightByHash(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		height, err := chainInstance.GetSnapshotHeightByHash(snapshotBlock.Hash)
		if err != nil {
			panic(err)
		}
		if height != snapshotBlock.Height {
			panic("error")
		}
	}

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))
		if err != nil {
			panic(err)
		}
		height, err := chainInstance.GetSnapshotHeightByHash(hash)
		if err != nil {
			panic(err)
		}
		if height > 0 {
			panic("error")
		}
	}

}

func GetSnapshotBlockByHeight(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(snapshotBlock, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHeight(snapshotBlock.Height)
			} else {
				return chainInstance.GetSnapshotBlockByHeight(snapshotBlock.Height)
			}
		}, onlyHeader)
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	for h := uint64(latestSnapshotBlock.Height + 1); h <= latestSnapshotBlock.Height+100; h++ {

		checkSnapshotBlock(nil, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHeight(h)
			} else {
				return chainInstance.GetSnapshotBlockByHeight(h)
			}
		}, onlyHeader)
	}
}

func GetSnapshotBlockByHash(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(snapshotBlock, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHash(snapshotBlock.Hash)
			} else {
				return chainInstance.GetSnapshotBlockByHash(snapshotBlock.Hash)
			}
		}, onlyHeader)
	}

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))

		if err != nil {
			panic(err)
		}

		checkSnapshotBlock(nil, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHash(hash)
			} else {
				return chainInstance.GetSnapshotBlockByHash(hash)
			}
		}, onlyHeader)
	}
}

func GetRangeSnapshotBlocks(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {

	checkSnapshotBlocks(snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		startHash := snapshotBlockList[startIndex].Hash
		endHash := snapshotBlockList[endIndex].Hash
		if onlyHeader {
			return chainInstance.GetRangeSnapshotHeaders(startHash, endHash)
		} else {
			return chainInstance.GetRangeSnapshotBlocks(startHash, endHash)
		}
	}, true, onlyHeader)

}

func GetSnapshotBlocks(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {

	checkSnapshotBlocks(snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		endHash := snapshotBlockList[endIndex].Hash
		if onlyHeader {
			return chainInstance.GetSnapshotHeaders(endHash, false, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocks(endHash, false, uint64(endIndex-startIndex+1))
		}
	}, false, onlyHeader)

	checkSnapshotBlocks(snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		startHash := snapshotBlockList[startIndex].Hash

		if onlyHeader {
			return chainInstance.GetSnapshotHeaders(startHash, true, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocks(startHash, true, uint64(endIndex-startIndex+1))
		}
	}, true, onlyHeader)
}

func GetSnapshotBlocksByHeight(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	checkSnapshotBlocks(snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		if onlyHeader {
			return chainInstance.GetSnapshotHeadersByHeight(snapshotBlockList[endIndex].Height, false, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocksByHeight(snapshotBlockList[endIndex].Height, false, uint64(endIndex-startIndex+1))
		}
	}, false, onlyHeader)

	checkSnapshotBlocks(snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		if onlyHeader {
			return chainInstance.GetSnapshotHeadersByHeight(snapshotBlockList[startIndex].Height, true, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocksByHeight(snapshotBlockList[startIndex].Height, true, uint64(endIndex-startIndex+1))
		}
	}, true, onlyHeader)
}

func GetConfirmSnapshotBlockByAbHash(chainInstance *chain, accounts map[types.Address]*Account, onlyHeader bool) {
	for _, account := range accounts {
		for snapshotHash, blockHashMap := range account.ConfirmedBlockMap {
			snapshotBlock, err := chainInstance.GetSnapshotBlockByHash(snapshotHash)
			if err != nil {
				panic(err)
			}
			if snapshotBlock == nil {
				panic("error")
			}

			for blockHash := range blockHashMap {
				checkSnapshotBlock(snapshotBlock, func() (*ledger.SnapshotBlock, error) {
					if onlyHeader {
						return chainInstance.GetConfirmSnapshotHeaderByAbHash(blockHash)
					} else {
						return chainInstance.GetConfirmSnapshotBlockByAbHash(blockHash)
					}
				}, onlyHeader)
			}

		}

	}
}

func QueryLatestSnapshotBlock(chainInstance *chain) {
	latestSnapshotBlock, err := chainInstance.QueryLatestSnapshotBlock()
	if err != nil {
		panic(err)
	}
	if latestSnapshotBlock == nil {
		panic(err)
	}

	checkSnapshotBlock(latestSnapshotBlock, func() (*ledger.SnapshotBlock, error) {
		return chainInstance.GetLatestSnapshotBlock(), nil
	}, false)

	latestHeight := latestSnapshotBlock.Height
	for h := latestHeight + 1; h < latestHeight+100; h++ {
		checkSnapshotBlock(nil, func() (*ledger.SnapshotBlock, error) {
			return chainInstance.GetSnapshotBlockByHeight(h)
		}, false)
	}
}

func GetSnapshotHeaderBeforeTime(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(snapshotBlock, func() (*ledger.SnapshotBlock, error) {
			time := snapshotBlock.Timestamp.Add(time.Second)
			return chainInstance.GetSnapshotHeaderBeforeTime(&time)
		}, true)
	}

	lastSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	for i := 1; i <= 100; i++ {
		checkSnapshotBlock(lastSnapshotBlock, func() (*ledger.SnapshotBlock, error) {
			futureTime := lastSnapshotBlock.Timestamp.Add(2 * time.Duration(i) * time.Second)

			return chainInstance.GetSnapshotHeaderBeforeTime(&futureTime)
		}, true)
	}

	genesisSnapshotBlock := chainInstance.GetGenesisSnapshotBlock()

	for i := 1; i <= 100; i++ {
		checkSnapshotBlock(nil, func() (*ledger.SnapshotBlock, error) {
			pastTime := genesisSnapshotBlock.Timestamp.Add(-2 * time.Duration(i) * time.Second)

			return chainInstance.GetSnapshotHeaderBeforeTime(&pastTime)
		}, true)
	}

}

func GetSnapshotHeadersAfterOrEqualTime(chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	if len(snapshotBlockList) <= 0 {
		return
	}

	lastSnapshotBlock := snapshotBlockList[len(snapshotBlockList)-1]
	for _, snapshotBlock := range snapshotBlockList {

		snapshotBlocks, err := chainInstance.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{
			Height: lastSnapshotBlock.Height,
			Hash:   lastSnapshotBlock.Hash,
		}, snapshotBlock.Timestamp, nil)

		if err != nil {
			panic(err)
		}
		for i, querySnapshotBlock := range snapshotBlocks {
			if querySnapshotBlock.Hash != snapshotBlockList[len(snapshotBlockList)-1-i].Hash {
				panic("error")
			}
		}
	}
}

func GetSubLedger(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	checkSubLedger(chainInstance, accounts, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotChunk, error) {
		return chainInstance.GetSubLedger(snapshotBlockList[startIndex].Height, snapshotBlockList[endIndex].Height)
	})
}

func GetSubLedgerAfterHeight(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	checkSubLedger(chainInstance, accounts, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotChunk, error) {
		return chainInstance.GetSubLedgerAfterHeight(snapshotBlockList[startIndex].Height)
	})

}

func checkSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, queryBlock func() (*ledger.SnapshotBlock, error), onlyHeader bool) {
	querySnapshotBlock, err := queryBlock()
	if err != nil {
		panic(err)
	}
	if snapshotBlock == nil {
		if querySnapshotBlock == nil {
			return
		} else {
			panic(fmt.Sprintf("%+v\n", querySnapshotBlock))
		}
	} else {
		if querySnapshotBlock == nil {
			panic(fmt.Sprintf("snapshot block: %+v\n", snapshotBlock))
		}
	}

	if snapshotBlock != nil && querySnapshotBlock != nil && querySnapshotBlock.Hash != snapshotBlock.Hash {
		panic(fmt.Sprintf("querySnapshotBlock: %+v\n, snapshotBlock: %+v\n", querySnapshotBlock, snapshotBlock))
	}
	if !onlyHeader {
		if len(querySnapshotBlock.SnapshotContent) != len(snapshotBlock.SnapshotContent) {
			panic("error")
		}

		for addr, hashHeight := range querySnapshotBlock.SnapshotContent {
			hashHeight2 := snapshotBlock.SnapshotContent[addr]
			if hashHeight.Hash != hashHeight2.Hash {
				panic("error")
			}

			if hashHeight.Height != hashHeight2.Height {
				panic("error")
			}
		}
	}
}

func checkSnapshotBlocks(snapshotBlockList []*ledger.SnapshotBlock,
	getBlocks func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error), higher bool, onlyHeader bool) {

	sbListLen := len(snapshotBlockList)
	var start, end int

	for {
		start = end + 1
		end = start + 9
		if start >= sbListLen {
			break
		}

		if end >= sbListLen {
			end = sbListLen - 1
		}

		blocks, err := getBlocks(start, end)
		if err != nil {
			panic(end)
		}

		blocksLen := len(blocks)
		if blocksLen != (end - start + 1) {
			panic("error")
		}

		var prevBlock *ledger.SnapshotBlock
		var block *ledger.SnapshotBlock

		index := 0
		for index < blocksLen {
			if higher {
				block = blocks[index]
			} else {
				block = blocks[blocksLen-index-1]
			}
			snapshotBlock := snapshotBlockList[start+index]

			if prevBlock != nil && (prevBlock.Hash != block.PrevHash || prevBlock.Height != block.Height-1) {
				panic(fmt.Sprintf("prevBlock is %+v\n, block is %+v\n", prevBlock, block))
			}
			if snapshotBlockList[start+index].Hash != block.Hash {
				panic(fmt.Sprintf("block is %+v\n, snapshotBlock is %+v\n", block, snapshotBlockList[start+index]))
			}
			if !onlyHeader {
				if len(block.SnapshotContent) != len(snapshotBlock.SnapshotContent) {
					panic("error")
				}

				for addr, hashHeight := range block.SnapshotContent {
					hashHeight2 := snapshotBlock.SnapshotContent[addr]
					if hashHeight.Hash != hashHeight2.Hash {
						panic("error")
					}

					if hashHeight.Height != hashHeight2.Height {
						panic("error")
					}
				}
			}

			index++
			prevBlock = block
		}
	}
}

func checkSubLedger(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock,
	getSubLedger func(startIndex, endIndex int) ([]*ledger.SnapshotChunk, error)) {

	var start, end int

	sbListLen := len(snapshotBlockList)

	for {
		start = end + 1
		end = start + 9

		if start >= sbListLen {
			break
		}

		if end >= sbListLen {
			end = sbListLen - 1
		}

		segs, err := getSubLedger(start, end)
		if err != nil {
			panic(err)
		}

		startSnapshotBlock := snapshotBlockList[start]
		for i := 0; i < len(segs); i++ {
			seg := segs[i]
			snapshotBlock := seg.SnapshotBlock

			checkSnapshotBlock(snapshotBlock, func() (*ledger.SnapshotBlock, error) {
				return chainInstance.GetSnapshotBlockByHeight(startSnapshotBlock.Height + uint64(i))
			}, false)

			accountBlocks := seg.AccountBlocks
			if i == 0 {
				if len(accountBlocks) > 0 {
					panic("error")
				}
				continue
			}

			blockHashMap := make(map[types.Hash]struct{}, len(accountBlocks))
			for _, accountBlock := range accountBlocks {
				blockHashMap[accountBlock.Hash] = struct{}{}
			}

			if snapshotBlock != nil {
				for _, account := range accounts {
					confirmedBlockHashList := account.ConfirmedBlockMap[snapshotBlock.Hash]
					for hash := range confirmedBlockHashList {
						if _, ok := blockHashMap[hash]; ok {
							delete(blockHashMap, hash)
						} else {
							panic("error")
						}
					}
				}

			} else {
				for _, account := range accounts {
					for hash := range account.UnconfirmedBlocks {
						if _, ok := blockHashMap[hash]; ok {
							delete(blockHashMap, hash)
						} else {
							panic("error")
						}
					}
				}
			}

			if len(blockHashMap) > 0 {
				for hash := range blockHashMap {
					block, err := chainInstance.GetAccountBlockByHash(hash)
					if err != nil {
						panic(err)
					}
					fmt.Printf("account block: %+v\n", block)

					confirmSnapshotBlock, err := chainInstance.GetConfirmSnapshotBlockByAbHash(hash)
					if err != nil {
						panic(err)
					}
					scStr := ""
					for addr, hashHeight := range confirmSnapshotBlock.SnapshotContent {
						scStr += fmt.Sprintf("%s %d %s, ", addr.String(), hashHeight.Height, hashHeight.Hash)
					}
					fmt.Printf("confirm snapshot block: %+v %s\n", confirmSnapshotBlock, scStr)
				}

				panic(fmt.Sprintf("blockHashMap: %+v\n snapshotBlock: %+v\n", blockHashMap, snapshotBlock))
			}
		}
	}
}

func testSnapshotBlock(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("IsGenesisSnapshotBlock", func(t *testing.T) {
		IsGenesisSnapshotBlock(chainInstance)
	})
	t.Run("IsSnapshotBlockExisted", func(t *testing.T) {
		IsSnapshotBlockExisted(chainInstance, snapshotBlockList)
	})
	t.Run("GetGenesisSnapshotBlock", func(t *testing.T) {
		GetGenesisSnapshotBlock(chainInstance)
	})
	t.Run("GetLatestSnapshotBlock", func(t *testing.T) {
		GetLatestSnapshotBlock(chainInstance)
	})
	t.Run("GetSnapshotHeightByHash", func(t *testing.T) {
		GetSnapshotHeightByHash(chainInstance, snapshotBlockList)
	})
	t.Run("GetSnapshotHeaderByHeight", func(t *testing.T) {
		GetSnapshotBlockByHeight(chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlockByHeight", func(t *testing.T) {
		GetSnapshotBlockByHeight(chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeaderByHash", func(t *testing.T) {
		GetSnapshotBlockByHash(chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlockByHash", func(t *testing.T) {
		GetSnapshotBlockByHash(chainInstance, snapshotBlockList, false)
	})

	t.Run("GetRangeSnapshotHeaders", func(t *testing.T) {
		GetRangeSnapshotBlocks(chainInstance, snapshotBlockList, true)
	})

	t.Run("GetRangeSnapshotBlocks", func(t *testing.T) {
		GetRangeSnapshotBlocks(chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeaders", func(t *testing.T) {
		GetSnapshotBlocks(chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlocks", func(t *testing.T) {
		GetSnapshotBlocks(chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeadersByHeight", func(t *testing.T) {
		GetSnapshotBlocksByHeight(chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlocksByHeight", func(t *testing.T) {
		GetSnapshotBlocksByHeight(chainInstance, snapshotBlockList, false)
	})

	t.Run("GetConfirmSnapshotHeaderByAbHash", func(t *testing.T) {
		GetConfirmSnapshotBlockByAbHash(chainInstance, accounts, true)
	})

	t.Run("GetConfirmSnapshotBlockByAbHash", func(t *testing.T) {
		GetConfirmSnapshotBlockByAbHash(chainInstance, accounts, false)
	})
	t.Run("QueryLatestSnapshotBlock", func(t *testing.T) {
		QueryLatestSnapshotBlock(chainInstance)
	})

	t.Run("GetSnapshotHeaderBeforeTime", func(t *testing.T) {
		GetSnapshotHeaderBeforeTime(chainInstance, snapshotBlockList)
	})

	t.Run("GetSnapshotHeadersAfterOrEqualTime", func(t *testing.T) {
		GetSnapshotHeadersAfterOrEqualTime(chainInstance, snapshotBlockList)
	})

	t.Run("GetSubLedger", func(t *testing.T) {
		GetSubLedger(chainInstance, accounts, snapshotBlockList)
	})

	t.Run("GetSubLedgerAfterHeight", func(t *testing.T) {
		GetSubLedgerAfterHeight(chainInstance, accounts, snapshotBlockList)
	})

}

func testSnapshotBlockNoTesting(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {

	IsGenesisSnapshotBlock(chainInstance)

	IsSnapshotBlockExisted(chainInstance, snapshotBlockList)

	GetGenesisSnapshotBlock(chainInstance)

	GetLatestSnapshotBlock(chainInstance)

	GetSnapshotHeightByHash(chainInstance, snapshotBlockList)

	GetSnapshotBlockByHeight(chainInstance, snapshotBlockList, true)

	GetSnapshotBlockByHeight(chainInstance, snapshotBlockList, false)

	GetSnapshotBlockByHash(chainInstance, snapshotBlockList, true)

	GetSnapshotBlockByHash(chainInstance, snapshotBlockList, false)

	GetRangeSnapshotBlocks(chainInstance, snapshotBlockList, true)

	GetRangeSnapshotBlocks(chainInstance, snapshotBlockList, false)

	GetSnapshotBlocks(chainInstance, snapshotBlockList, true)

	GetSnapshotBlocks(chainInstance, snapshotBlockList, false)

	GetSnapshotBlocksByHeight(chainInstance, snapshotBlockList, true)

	GetSnapshotBlocksByHeight(chainInstance, snapshotBlockList, false)

	GetConfirmSnapshotBlockByAbHash(chainInstance, accounts, true)

	GetConfirmSnapshotBlockByAbHash(chainInstance, accounts, false)

	QueryLatestSnapshotBlock(chainInstance)

	GetSnapshotHeaderBeforeTime(chainInstance, snapshotBlockList)

	GetSnapshotHeadersAfterOrEqualTime(chainInstance, snapshotBlockList)

	GetSubLedger(chainInstance, accounts, snapshotBlockList)

	GetSubLedgerAfterHeight(chainInstance, accounts, snapshotBlockList)

}
