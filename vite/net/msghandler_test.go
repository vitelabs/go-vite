package net

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func randInt(m, n int) int {
	r := rand.Intn(n - m)
	return r + m
}

func mockAccountMap(addrm, addrn, blockm, blockn int) (ret accountBlockMap, total int) {
	ret = make(accountBlockMap)
	accountCount := randInt(addrm, addrn)
	var addr types.Address
	for i := 0; i < accountCount; i++ {
		crand.Read(addr[:])
		count := randInt(blockm, blockn)
		ret[addr] = make([]*ledger.AccountBlock, count)
		total += count
	}

	return ret, total
}

func Test_SplitAccountMap(t *testing.T) {
	mblocks, total := mockAccountMap(100, 1000, 100, 1000)
	total2 := countAccountBlocks(mblocks)
	if uint64(total) != total2 {
		t.Fail()
	} else {
		fmt.Println("countAccountBlocks right")
	}

	matrix := splitAccountMap(mblocks)
	shouldLen := total/1000 + 1
	if len(matrix) != shouldLen {
		t.Fail()
	} else {
		fmt.Println("splitAccountMap length right")
	}

	total3 := 0
	for _, blocks := range matrix {
		total3 += len(blocks)
	}

	if total != total3 {
		t.Fail()
	} else {
		fmt.Println("splitAccountMap total right")
	}

	fmt.Println(total, total2, total3)
}

func Test_SplitAccountMap_Min(t *testing.T) {
	mblocks, total := mockAccountMap(100, 300, 1, 2)
	total2 := countAccountBlocks(mblocks)
	if uint64(total) != total2 {
		t.Fail()
	} else {
		fmt.Println("countAccountBlocks right")
	}

	matrix := splitAccountMap(mblocks)
	shouldLen := total/1000 + 1
	if len(matrix) != shouldLen {
		t.Fail()
	} else {
		fmt.Println("splitAccountMap length right")
	}

	total3 := 0
	for _, blocks := range matrix {
		total3 += len(blocks)
	}

	if total != total3 {
		t.Fail()
	} else {
		fmt.Println("splitAccountMap total right")
	}

	fmt.Println(total, total2, total3)
}

type mockSnapshotReader struct {
	blocksByHash   map[types.Hash]*ledger.SnapshotBlock
	blocksByHeight map[uint64]*ledger.SnapshotBlock
}

func (m *mockSnapshotReader) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return m.blocksByHeight[height], nil
}

func (m *mockSnapshotReader) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	return m.blocksByHash[hash], nil
}

func (m *mockSnapshotReader) GetSnapshotBlocks(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (m *mockSnapshotReader) GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func mockHash() (hash types.Hash) {
	_, _ = rand.Read(hash[:])
	return
}

func TestCheckHandler(t *testing.T) {
	const to = 600
	hhs := []*ledger.HashHeight{
		{75, mockHash()},
		{175, mockHash()},
	}

	var check = &message.HashHeightList{
		Points: hhs,
	}

	chain := &mockSnapshotReader{
		blocksByHash:   make(map[types.Hash]*ledger.SnapshotBlock),
		blocksByHeight: make(map[uint64]*ledger.SnapshotBlock),
	}
	for _, hh := range hhs {
		block := &ledger.SnapshotBlock{
			Hash:   hh.Hash,
			Height: hh.Height,
		}

		chain.blocksByHash[hh.Hash] = block
		chain.blocksByHeight[hh.Height] = block
	}

	for start := uint64(0); start < to; start += step {
		block := &ledger.SnapshotBlock{
			Hash:   mockHash(),
			Height: start,
		}
		chain.blocksByHash[block.Hash] = block
		chain.blocksByHeight[block.Height] = block
	}

	var checkH = &checkHandler{
		chain: &mockSnapshotReader{
			blocksByHash:   nil,
			blocksByHeight: nil,
		},
		log: netLog,
	}

	cd, payload := checkH.handleCheck(check)
	if cd != CodeCheckResult {
		t.Errorf("error code: %d", cd)
	} else {
		checkResult := payload.(*ledger.HashHeight)

		if checkResult.Height != 175 || checkResult.Hash != hhs[1].Hash {
			t.Error("wrong fep")
		}
	}
}
