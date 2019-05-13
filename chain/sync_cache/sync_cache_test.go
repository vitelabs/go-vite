package sync_cache

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func TestSyncCacheLoad(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = path.Join(dir, "sync_cache")
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		panic(err)
	}

	filenames := []string{
		"f_2_0100000000000000000000000000000000000000000000000000000000000000_100_0200000000000000000000000000000000000000000000000000000000000000",
		"f_101_0300000000000000000000000000000000000000000000000000000000000000_200_0400000000000000000000000000000000000000000000000000000000000000",
		"t_201_0500000000000000000000000000000000000000000000000000000000000000_300_0600000000000000000000000000000000000000000000000000000000000000",
		"f_301_0700000000000000000000000000000000000000000000000000000000000000_400_0800000000000000000000000000000000000000000000000000000000000000.v",
	}

	var f *os.File
	for _, file := range filenames {
		f, err = os.Create(path.Join(dir, file))
		if err != nil {
			panic(fmt.Errorf("failed to create segment file: %v", err))
		}
		_ = f.Close()
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs := cache.Chunks()
	cs2 := interfaces.SegmentList{
		{
			Bound:    [2]uint64{2, 100},
			PrevHash: types.Hash{1},
			Hash:     types.Hash{2},
		},
		{
			Bound:    [2]uint64{101, 200},
			PrevHash: types.Hash{3},
			Hash:     types.Hash{4},
		},
		{
			Bound:    [2]uint64{301, 400},
			PrevHash: types.Hash{7},
			Hash:     types.Hash{8},
		},
	}

	for i, c := range cs {
		if c != cs2[i] {
			t.Error("wrong segment")
		}
	}

	reader, err := cache.NewReader(cs2[0])
	if err != nil {
		panic(err)
	}
	if true == reader.Verified() {
		t.Error("should not verified")
	}
	err = reader.Close()
	if err != nil {
		panic(err)
	}
	reader.Verify()

	reader, err = cache.NewReader(cs2[0])
	if err != nil {
		panic(err)
	}
	if false == reader.Verified() {
		t.Error("should verified")
	}
	err = reader.Close()
	if err != nil {
		panic(err)
	}

	reader, err = cache.NewReader(cs2[2])
	if err != nil {
		panic(err)
	}
	if false == reader.Verified() {
		t.Error("should verified")
	}
	err = reader.Close()
	if err != nil {
		panic(err)
	}
	reader.Verify()
	reader, err = cache.NewReader(cs2[2])
	if err != nil {
		panic(err)
	}
	if false == reader.Verified() {
		t.Error("should verified")
	}
	err = reader.Close()
	if err != nil {
		panic(err)
	}

	// create writer
	seg := interfaces.Segment{
		Bound:    [2]uint64{401, 500},
		Hash:     types.Hash{9},
		PrevHash: types.Hash{10},
	}
	w, err := cache.NewWriter(seg)
	if err != nil {
		panic(err)
	}
	cs = cache.Chunks()
	for _, c := range cs {
		if c == seg {
			t.Error("write not close, should not in chunk list")
		}
	}

	_, err = w.Write([]byte("hello"))
	if err != nil {
		panic(err)
	}
	err = w.Close()
	if err != nil {
		panic(err)
	}

	var in bool
	cs = cache.Chunks()
	for _, c := range cs {
		if c == seg {
			in = true
		}
	}
	if false == in {
		t.Error("write close, should in chunk list")
	}
}
