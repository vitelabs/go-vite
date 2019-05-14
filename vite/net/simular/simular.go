package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/interfaces"
)

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir1 := filepath.Join(dir, ".mock1")
	c1, err := newChain(dir1)
	if err != nil {
		panic(err)
	}

	dir2 := filepath.Join(dir, ".mock2")
	_ = os.RemoveAll(dir2)
	c2, err := newChain(dir2)
	if err != nil {
		panic(err)
	}

	s2Current := c2.GetLatestSnapshotBlock()
	prevHeight, prevHash := s2Current.Height, s2Current.Hash

	start := s2Current.Height
	end := c1.GetLatestSnapshotBlock().Height

	from := start + 1
	for from <= end {
		to := from + 100 - 1
		if to > end {
			to = end
		}

		fmt.Printf("download chunk %d-%d\n", from, to)

		var r interfaces.LedgerReader
		r, err = c1.GetLedgerReaderByHeight(from, to)
		if err != nil {
			panic(err)
		}

		var w io.WriteCloser
		w, err = c2.GetSyncCache().NewWriter(r.Seg())
		if err != nil {
			fmt.Println(c2.GetSyncCache().Chunks())
			panic(err)
		}

		var n int64
		n, err = io.Copy(w, r)

		if n != int64(r.Size()) {
			panic("size not equal")
		} else {
			fmt.Printf("write cache %d-%d %d bytes\n", from, to, n)
		}

		_ = r.Close()
		_ = w.Close()

		var r2 interfaces.ReadCloser
		r2, err = c2.GetSyncCache().NewReader(r.Seg())
		if err != nil {
			panic(err)
		} else {
			fmt.Printf("read cache %d-%d\n", from, to)
		}

		for {
			var sb *ledger.SnapshotBlock
			_, sb, err = r2.Read()
			if err != nil {
				if err != io.EOF {
					panic(err)
				} else {
					break
				}
			} else if sb != nil {
				if sb.PrevHash != prevHash || sb.Height != prevHeight+1 {
					fmt.Printf("%d/%s %d/%s/%s\n", prevHeight, prevHash, sb.Height, sb.Hash, sb.PrevHash)
					panic("not continuous")
				} else {
					prevHash = sb.Hash
					prevHeight = sb.Height
					fmt.Printf("snapshot %d\n", sb.Height)
				}
			}
		}

		from = to + 1
	}
}
