package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	s1 := newSyncer(filepath.Join(dir, ".mock1"))
	s2 := newSyncer(filepath.Join(dir, ".mock2"))

	fmt.Println("mock1", s1.height(), s1.chain.GetSyncCache().Chunks())
	fmt.Println("mock2", s2.height(), s2.chain.GetSyncCache().Chunks())

	s2.from = s2.height() + 1
	s2.to = s1.height()

	var from, to uint64
	var w io.WriteCloser
	for from = s2.from; from <= s2.to; from = to + 1 {
		to = from + chunkSize
		if to > s2.to {
			to = s2.to
		}

		fmt.Println("download", from, to)
		w, err = s2.chain.GetSyncCache().NewWriter(from, to)
		if err != nil {
			panic(err)
		}

		err = s1.request(from, to, w)
		if err != nil {
			panic(err)
		}

		_ = w.Close()
	}

	s2.readCacheLoop()

	time.Sleep(time.Minute)
}
