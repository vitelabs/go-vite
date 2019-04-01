package chain

import (
	"encoding/binary"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/genesis"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
	"time"
)

func TestChain_SnapshotBlock(t *testing.T) {
	chainInstance, accounts, _, _, _, snapshotBlockList := SetUp(t, 10, 1234, 7)

	testSnapshotBlock(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func IsSnapshotBlockExisted(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		result, err := chainInstance.IsSnapshotBlockExisted(snapshotBlock.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if !result {
			t.Fatal("error")
		}
	}

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))
		if err != nil {
			t.Fatal(err)
		}
		result, err := chainInstance.IsSnapshotBlockExisted(hash)
		if err != nil {
			t.Fatal(err)
		}
		if result {
			t.Fatal("error")
		}
	}

}
func GetGenesisSnapshotBlock(t *testing.T, chainInstance *chain) {
	genesisSnapshotBlock := chainInstance.GetGenesisSnapshotBlock()

	correctFirstSb := chain_genesis.NewGenesisSnapshotBlock()
	if genesisSnapshotBlock.Hash != correctFirstSb.Hash {
		t.Fatal("error")
	}
}

func GetLatestSnapshotBlock(t *testing.T, chainInstance *chain) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	nextSb, err := chainInstance.GetSnapshotHeaderByHeight(latestSnapshotBlock.Height + 1)
	if err != nil {
		t.Fatal(err)
	}
	if nextSb != nil {
		t.Fatal("error")
	}
}

func GetSnapshotHeightByHash(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		height, err := chainInstance.GetSnapshotHeightByHash(snapshotBlock.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if height != snapshotBlock.Height {
			t.Fatal("error")
		}
	}

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))
		if err != nil {
			t.Fatal(err)
		}
		height, err := chainInstance.GetSnapshotHeightByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if height > 0 {
			t.Fatal("error")
		}
	}

}

func GetSnapshotBlockByHeight(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(t, snapshotBlock, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHeight(snapshotBlock.Height)
			} else {
				return chainInstance.GetSnapshotBlockByHeight(snapshotBlock.Height)
			}
		}, onlyHeader)
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	for h := uint64(latestSnapshotBlock.Height + 1); h <= latestSnapshotBlock.Height+100; h++ {

		checkSnapshotBlock(t, nil, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHeight(h)
			} else {
				return chainInstance.GetSnapshotBlockByHeight(h)
			}
		}, onlyHeader)
	}
}

func GetSnapshotBlockByHash(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(t, snapshotBlock, func() (*ledger.SnapshotBlock, error) {
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
			t.Fatal(err)
		}

		checkSnapshotBlock(t, nil, func() (*ledger.SnapshotBlock, error) {
			if onlyHeader {
				return chainInstance.GetSnapshotHeaderByHash(hash)
			} else {
				return chainInstance.GetSnapshotBlockByHash(hash)
			}
		}, onlyHeader)
	}
}

func GetRangeSnapshotBlocks(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {

	checkSnapshotBlocks(t, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		startHash := snapshotBlockList[startIndex].Hash
		endHash := snapshotBlockList[endIndex].Hash
		if onlyHeader {
			return chainInstance.GetRangeSnapshotHeaders(startHash, endHash)
		} else {
			return chainInstance.GetRangeSnapshotBlocks(startHash, endHash)
		}
	}, true, onlyHeader)

}

func GetSnapshotBlocks(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {

	checkSnapshotBlocks(t, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		endHash := snapshotBlockList[endIndex].Hash
		if onlyHeader {
			return chainInstance.GetSnapshotHeaders(endHash, false, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocks(endHash, false, uint64(endIndex-startIndex+1))
		}
	}, false, onlyHeader)

	checkSnapshotBlocks(t, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		startHash := snapshotBlockList[startIndex].Hash

		if onlyHeader {
			return chainInstance.GetSnapshotHeaders(startHash, true, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocks(startHash, true, uint64(endIndex-startIndex+1))
		}
	}, true, onlyHeader)
}

func GetSnapshotBlocksByHeight(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock, onlyHeader bool) {
	checkSnapshotBlocks(t, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		if onlyHeader {
			return chainInstance.GetSnapshotHeadersByHeight(snapshotBlockList[endIndex].Height, false, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocksByHeight(snapshotBlockList[endIndex].Height, false, uint64(endIndex-startIndex+1))
		}
	}, false, onlyHeader)

	checkSnapshotBlocks(t, snapshotBlockList, func(startIndex, endIndex int) ([]*ledger.SnapshotBlock, error) {
		if onlyHeader {
			return chainInstance.GetSnapshotHeadersByHeight(snapshotBlockList[startIndex].Height, true, uint64(endIndex-startIndex+1))
		} else {
			return chainInstance.GetSnapshotBlocksByHeight(snapshotBlockList[startIndex].Height, true, uint64(endIndex-startIndex+1))
		}
	}, true, onlyHeader)
}

func GetConfirmSnapshotBlockByAbHash(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, onlyHeader bool) {
	for _, account := range accounts {
		for snapshotHash, blockHashMap := range account.ConfirmedBlockMap {
			snapshotBlock, err := chainInstance.GetSnapshotBlockByHash(snapshotHash)
			if err != nil {
				t.Fatal(err)
			}
			if snapshotBlock == nil {
				t.Fatal("error")
			}

			for blockHash := range blockHashMap {
				checkSnapshotBlock(t, snapshotBlock, func() (*ledger.SnapshotBlock, error) {
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

func QueryLatestSnapshotBlock(t *testing.T, chainInstance *chain) {
	latestSnapshotBlock, err := chainInstance.QueryLatestSnapshotBlock()
	if err != nil {
		t.Fatal(err)
	}
	if latestSnapshotBlock == nil {
		t.Fatal(err)
	}

	checkSnapshotBlock(t, latestSnapshotBlock, func() (*ledger.SnapshotBlock, error) {
		return chainInstance.GetLatestSnapshotBlock(), nil
	}, false)

	latestHeight := latestSnapshotBlock.Height
	for h := latestHeight + 1; h < latestHeight+100; h++ {
		checkSnapshotBlock(t, nil, func() (*ledger.SnapshotBlock, error) {
			return chainInstance.GetSnapshotBlockByHeight(h)
		}, false)
	}
}

func GetSnapshotHeaderBeforeTime(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		checkSnapshotBlock(t, snapshotBlock, func() (*ledger.SnapshotBlock, error) {
			time := snapshotBlock.Timestamp.Add(time.Second)
			return chainInstance.GetSnapshotHeaderBeforeTime(&time)
		}, true)
	}
}

func GetSnapshotHeadersAfterOrEqualTime(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	lastSnapshotBlock := snapshotBlockList[len(snapshotBlockList)-1]
	for _, snapshotBlock := range snapshotBlockList {

		snapshotBlocks, err := chainInstance.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{
			Height: lastSnapshotBlock.Height,
			Hash:   lastSnapshotBlock.Hash,
		}, snapshotBlock.Timestamp, nil)

		if err != nil {
			t.Fatal(err)
		}
		for i, querySnapshotBlock := range snapshotBlocks {
			if querySnapshotBlock.Hash != snapshotBlockList[len(snapshotBlockList)-1-i].Hash {
				t.Fatal("error")
			}
		}
	}
}

func GetSubLedger(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	checkSubLedger(t, chainInstance, accounts, snapshotBlockList, func(startIndex, endIndex int) ([]*chain_block.SnapshotSegment, error) {
		return chainInstance.GetSubLedger(snapshotBlockList[startIndex].Height, snapshotBlockList[endIndex].Height)
	})
}

func GetSubLedgerAfterHeight(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	checkSubLedger(t, chainInstance, accounts, snapshotBlockList, func(startIndex, endIndex int) ([]*chain_block.SnapshotSegment, error) {
		return chainInstance.GetSubLedgerAfterHeight(snapshotBlockList[startIndex].Height)
	})

}

func checkSnapshotBlock(t *testing.T, snapshotBlock *ledger.SnapshotBlock, queryBlock func() (*ledger.SnapshotBlock, error), onlyHeader bool) {
	querySnapshotBlock, err := queryBlock()
	if err != nil {
		t.Fatal(err)
	}
	if snapshotBlock == nil && querySnapshotBlock == nil {
		return
	}

	if querySnapshotBlock.Hash != snapshotBlock.Hash {
		t.Fatal(fmt.Sprintf("querySnapshotBlock: %+v\n, snapshotBlock: %+v\n", querySnapshotBlock, snapshotBlock))
	}
	if !onlyHeader {
		if len(querySnapshotBlock.SnapshotContent) != len(snapshotBlock.SnapshotContent) {
			t.Fatal("error")
		}

		for addr, hashHeight := range querySnapshotBlock.SnapshotContent {
			hashHeight2 := snapshotBlock.SnapshotContent[addr]
			if hashHeight.Hash != hashHeight2.Hash {
				t.Fatal("error")
			}

			if hashHeight.Height != hashHeight2.Height {
				t.Fatal("error")
			}
		}
	}
}

func checkSnapshotBlocks(t *testing.T, snapshotBlockList []*ledger.SnapshotBlock,
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
			t.Fatal(end)
		}

		blocksLen := len(blocks)
		if blocksLen != (end - start + 1) {
			t.Fatal("error")
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
				t.Fatal(fmt.Sprintf("prevBlock is %+v\n, block is %+v\n", prevBlock, block))
			}
			if snapshotBlockList[start+index].Hash != block.Hash {
				t.Fatal(fmt.Sprintf("block is %+v\n, snapshotBlock is %+v\n", block, snapshotBlockList[start+index]))
			}
			if !onlyHeader {
				if len(block.SnapshotContent) != len(snapshotBlock.SnapshotContent) {
					t.Fatal("error")
				}

				for addr, hashHeight := range block.SnapshotContent {
					hashHeight2 := snapshotBlock.SnapshotContent[addr]
					if hashHeight.Hash != hashHeight2.Hash {
						t.Fatal("error")
					}

					if hashHeight.Height != hashHeight2.Height {
						t.Fatal("error")
					}
				}
			}

			index++
			prevBlock = block
		}
	}
}

func checkSubLedger(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock,
	getSubLedger func(startIndex, endIndex int) ([]*chain_block.SnapshotSegment, error)) {

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
			t.Fatal(err)
		}

		for i := 0; i < len(segs); i++ {
			seg := segs[i]
			snapshotBlock := seg.SnapshotBlock

			checkSnapshotBlock(t, snapshotBlock, func() (*ledger.SnapshotBlock, error) {
				return chainInstance.GetSnapshotBlockByHeight(snapshotBlockList[start+i].Height)
			}, false)

			accountBlocks := seg.AccountBlocks
			if i == 0 {
				if len(accountBlocks) > 0 {
					t.Fatal("error")
				}
				continue
			}

			blockHashMap := make(map[types.Hash]struct{}, len(accountBlocks))
			for _, accountBlock := range accountBlocks {
				blockHashMap[accountBlock.Hash] = struct{}{}
			}

			if snapshotBlock != nil && snapshotBlock.Height != snapshotBlockList[start].Height {
				for _, account := range accounts {
					confirmedBlockHashList := account.ConfirmedBlockMap[snapshotBlock.Hash]
					for hash := range confirmedBlockHashList {
						if _, ok := blockHashMap[hash]; ok {
							delete(blockHashMap, hash)
						} else {
							t.Fatal("error")
						}
					}
				}
			} else if snapshotBlock == nil {
				for _, account := range accounts {
					for hash := range account.unconfirmedBlocks {
						if _, ok := blockHashMap[hash]; ok {
							delete(blockHashMap, hash)
						} else {
							t.Fatal("error")
						}
					}
				}
			}

			if len(blockHashMap) > 0 {
				for hash := range blockHashMap {
					block, err := chainInstance.GetAccountBlockByHash(hash)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Printf("account block: %+v\n", block)

					confirmSnapshotBlock, err := chainInstance.GetConfirmSnapshotBlockByAbHash(hash)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Printf("confirm snapshot block: %+v\n", confirmSnapshotBlock)
				}

				t.Fatal(fmt.Sprintf("blockHashMap: %+v\n snapshotBlock: %+v\n", blockHashMap, snapshotBlock))
			}
		}
	}
}

func testSnapshotBlock(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("IsSnapshotBlockExisted", func(t *testing.T) {
		IsSnapshotBlockExisted(t, chainInstance, snapshotBlockList)
	})
	t.Run("GetGenesisSnapshotBlock", func(t *testing.T) {
		GetGenesisSnapshotBlock(t, chainInstance)
	})
	t.Run("GetLatestSnapshotBlock", func(t *testing.T) {
		GetLatestSnapshotBlock(t, chainInstance)
	})
	t.Run("GetSnapshotHeightByHash", func(t *testing.T) {
		GetSnapshotHeightByHash(t, chainInstance, snapshotBlockList)
	})
	t.Run("GetSnapshotHeaderByHeight", func(t *testing.T) {
		GetSnapshotBlockByHeight(t, chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlockByHeight", func(t *testing.T) {
		GetSnapshotBlockByHeight(t, chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeaderByHash", func(t *testing.T) {
		GetSnapshotBlockByHash(t, chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlockByHash", func(t *testing.T) {
		GetSnapshotBlockByHash(t, chainInstance, snapshotBlockList, false)
	})

	t.Run("GetRangeSnapshotHeaders", func(t *testing.T) {
		GetRangeSnapshotBlocks(t, chainInstance, snapshotBlockList, true)
	})

	t.Run("GetRangeSnapshotBlocks", func(t *testing.T) {
		GetRangeSnapshotBlocks(t, chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeaders", func(t *testing.T) {
		GetSnapshotBlocks(t, chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlocks", func(t *testing.T) {
		GetSnapshotBlocks(t, chainInstance, snapshotBlockList, false)
	})

	t.Run("GetSnapshotHeadersByHeight", func(t *testing.T) {
		GetSnapshotBlocksByHeight(t, chainInstance, snapshotBlockList, true)
	})

	t.Run("GetSnapshotBlocksByHeight", func(t *testing.T) {
		GetSnapshotBlocksByHeight(t, chainInstance, snapshotBlockList, false)
	})

	t.Run("GetConfirmSnapshotHeaderByAbHash", func(t *testing.T) {
		GetConfirmSnapshotBlockByAbHash(t, chainInstance, accounts, true)
	})

	t.Run("GetConfirmSnapshotBlockByAbHash", func(t *testing.T) {
		GetConfirmSnapshotBlockByAbHash(t, chainInstance, accounts, false)
	})
	t.Run("QueryLatestSnapshotBlock", func(t *testing.T) {
		QueryLatestSnapshotBlock(t, chainInstance)
	})

	t.Run("GetSnapshotHeaderBeforeTime", func(t *testing.T) {
		GetSnapshotHeaderBeforeTime(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetSnapshotHeadersAfterOrEqualTime", func(t *testing.T) {
		GetSnapshotHeadersAfterOrEqualTime(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetSubLedger", func(t *testing.T) {
		GetSubLedger(t, chainInstance, accounts, snapshotBlockList)
	})

	t.Run("GetSubLedgerAfterHeight", func(t *testing.T) {
		GetSubLedgerAfterHeight(t, chainInstance, accounts, snapshotBlockList)
	})

}
