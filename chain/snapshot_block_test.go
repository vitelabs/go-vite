package chain

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/chain/genesis"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

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

func GetSnapshotHeaderByHeight(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		snapshotHeader, err := chainInstance.GetSnapshotHeaderByHeight(snapshotBlock.Height)
		if err != nil {
			t.Fatal(err)
		}
		if snapshotHeader.Hash != snapshotBlock.Hash {
			t.Fatal("error")
		}
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	for h := uint64(latestSnapshotBlock.Height + 1); h <= latestSnapshotBlock.Height+100; h++ {

		snapshotHeader, err := chainInstance.GetSnapshotHeaderByHeight(h)
		if err != nil {
			t.Fatal(err)
		}
		if snapshotHeader != nil {
			t.Fatal("error")
		}
	}
}

func GetSnapshotBlockByHeight(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		block, err := chainInstance.GetSnapshotBlockByHeight(snapshotBlock.Height)
		if err != nil {
			t.Fatal(err)
		}
		if block.Hash != snapshotBlock.Hash {
			t.Fatal("error")
		}
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

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	for h := uint64(latestSnapshotBlock.Height + 1); h <= latestSnapshotBlock.Height+100; h++ {

		block, err := chainInstance.GetSnapshotBlockByHeight(h)
		if err != nil {
			t.Fatal(err)
		}
		if block != nil {
			t.Fatal("error")
		}
	}
}

func GetSnapshotHeaderByHash(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		snapshotHeader, err := chainInstance.GetSnapshotHeaderByHash(snapshotBlock.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if snapshotHeader.Hash != snapshotBlock.Hash ||
			snapshotHeader.Height != snapshotBlock.Height {
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
		snapshotHeader, err := chainInstance.GetSnapshotHeaderByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if snapshotHeader != nil {
			t.Fatal("error")
		}
	}
}

func GetSnapshotBlockByHash(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		block, err := chainInstance.GetSnapshotBlockByHash(snapshotBlock.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if block.Hash != snapshotBlock.Hash {
			t.Fatal("error")
		}
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

	for i := uint64(1); i <= 100; i++ {
		uint64Bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(uint64Bytes, i)
		hash, err := types.BytesToHash(crypto.Hash256(uint64Bytes))
		if err != nil {
			t.Fatal(err)
		}
		snapshotBlock, err := chainInstance.GetSnapshotBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if snapshotBlock != nil {
			t.Fatal("error")
		}
	}
}

func GetRangeSnapshotHeaders(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	sbListLen := len(snapshotBlockList)
	var start, end int

	for end >= sbListLen {
		start = end + 1
		end = start + 9
		if end > sbListLen {
			end = sbListLen
		}

		startHash := snapshotBlockList[start].Hash
		endHash := snapshotBlockList[end].Hash
		blocks, err := chainInstance.GetRangeSnapshotHeaders(startHash, endHash)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != (end - start + 1) {
			t.Fatal("error")
		}
		var prevBlock *ledger.SnapshotBlock
		for index, block := range blocks {
			if prevBlock != nil && (prevBlock.Hash != block.PrevHash || prevBlock.Height != block.Height-1) {
				t.Fatal("error")
			}
			if snapshotBlockList[start+index].Hash != block.Hash {
				t.Fatal("error")
			}
			prevBlock = block
		}
	}

}

func GetRangeSnapshotBlocks(t *testing.T, chainInstance *chain, snapshotBlockList []*ledger.SnapshotBlock) {
	sbListLen := len(snapshotBlockList)
	var start, end int

	for end >= sbListLen {
		start = end + 1
		end = start + 9
		if end > sbListLen {
			end = sbListLen
		}

		startHash := snapshotBlockList[start].Hash
		endHash := snapshotBlockList[end].Hash
		blocks, err := chainInstance.GetRangeSnapshotBlocks(startHash, endHash)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != (end - start + 1) {
			t.Fatal("error")
		}
		var prevBlock *ledger.SnapshotBlock
		for index, block := range blocks {
			snapshotBlock := snapshotBlockList[start+index]
			if prevBlock != nil && (prevBlock.Hash != block.PrevHash || prevBlock.Height != block.Height-1) {
				t.Fatal("error")
			}
			if snapshotBlock.Hash != block.Hash {
				t.Fatal("error")
			}
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
			prevBlock = block
		}

	}

}
