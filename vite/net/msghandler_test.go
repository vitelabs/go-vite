package net

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"

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
