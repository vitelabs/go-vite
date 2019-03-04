package net

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/p2p"

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

func Test_SplitFiles(t *testing.T) {
	total := rand.Intn(9999)
	files := make([]*ledger.CompressedFileMeta, total)

	for i := 0; i < total; i++ {
		files[i] = &ledger.CompressedFileMeta{
			StartHeight: uint64(i),
			EndHeight:   uint64(i),
		}
	}

	const batch = 1000
	filess := splitFiles(files, batch)

	var start uint64
	var count int
	for k, fs := range filess {
		l := len(fs)
		if k != len(filess)-1 {
			if l != batch {
				t.Fatalf("files number should be %d, but get %d", batch, l)
			}
		} else {
			if l > batch {
				t.Fatalf("files number should small than %d, but get %d", batch, l)
			}
		}

		count += l

		for _, f := range fs {
			if f.StartHeight != start {
				t.Fatalf("file startHeight should be %d, but get %d", start, f.StartHeight)
			} else {
				start++
			}
		}
	}

	if count != total {
		t.Fatalf("files count should be %d, but get %d", total, count)
	}
}

type chain_getSubLedger struct {
}

func (c *chain_getSubLedger) GetSubLedgerByHeight(start, count uint64, forward bool) (fs []*ledger.CompressedFileMeta, cs [][2]uint64) {
	end := start + count - 1
	for i := start; i <= end; i++ {
		j := i + 3599
		if j > end {
			cs = append(cs, [2]uint64{i, end})
			return
		} else {
			fs = append(fs, &ledger.CompressedFileMeta{
				StartHeight:  i,
				EndHeight:    j,
				Filename:     "subgraph_" + strconv.FormatUint(i, 10) + "-" + strconv.FormatUint(j, 10),
				FileSize:     0,
				BlockNumbers: 3600,
			})
		}

		i = j
	}

	return
}

func (c *chain_getSubLedger) GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) (fs []*ledger.CompressedFileMeta, cs [][2]uint64, err error) {
	return
}

func Test_getSubLedgerHandler(t *testing.T) {
	var count = uint64(10000000)
	var from = uint64(1)
	var end = from + count - 1
	const forward = true

	counter := func() func(msgId uint64, payload p2p.Serializable) {
		var height = from
		return func(msgId uint64, payload p2p.Serializable) {
			fileList := payload.(*message.FileList)
			for _, f := range fileList.Files {
				if height+1 != f.StartHeight {
					t.Fatal("file is not continuous")
				}
				height = f.EndHeight
			}

			if len(fileList.Chunks) != 0 {
				c := fileList.Chunks[0]
				if height+1 != c[0] {
					t.Fatal("chunk is not continuous")
				}
				height = c[1]
				if height != (count + from - 1) {
					t.Fatal("height is too low")
				}
			}

			// last fileList
			if len(fileList.Files) < maxFilesOneTrip && height != end {
				t.Fatalf("height %d, not end %d\n", height, end)
			}
		}
	}

	mp := NewMockPeer()
	mp.Handlers[FileListCode] = counter()

	handler := &getSubLedgerHandler{
		chain: &chain_getSubLedger{},
	}

	msg, err := p2p.PackMsg(0, uint16(GetSubLedgerCode), 0, &message.GetSnapshotBlocks{
		From: ledger.HashHeight{
			Height: from,
		},
		Count:   count,
		Forward: forward,
	})

	if err != nil {
		t.Fatalf("pack query message error: %v\n", err)
	}

	err = handler.Handle(msg, mp)
	if err != nil {
		t.Fatalf("handle error: %v\n", err)
	}
}
