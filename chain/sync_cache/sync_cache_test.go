package sync_cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func equal(seg1, seg2 interfaces.Segment) error {
	if seg1.Bound != seg2.Bound {
		return fmt.Errorf("different bound: %v %v", seg1.Bound, seg2.Bound)
	}

	if seg1.Hash != seg2.Hash {
		return fmt.Errorf("different endHash: %s %s", seg1.Hash, seg2.Hash)
	}

	if seg1.PrevHash != seg2.PrevHash {
		return fmt.Errorf("different prevHash: %s %s", seg1.PrevHash, seg2.PrevHash)
	}

	return nil
}

func TestFilename(t *testing.T) {
	var seg = interfaces.Segment{
		Bound:    [2]uint64{101, 200},
		Hash:     types.Hash{},
		PrevHash: types.Hash{},
	}

	seg.Hash[len(seg.Hash)-1] = 1

	filename := toFilename(correctFilePrefix, seg)
	fmt.Println(filename)
	seg2, err := newSegmentByFilename(filename)
	if err != nil {
		panic(err)
	}

	err = equal(seg, seg2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoader(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = filepath.Join(dir, "mockr")
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		panic(err)
	}

	files := []string{
		"f_101_0000000000000000000000000000000000000000000000000000000000000000_200_0000000000000000000000000000000000000000000000000000000000000001",
		"t_201_0000000000000000000000000000000000000000000000000000000000000001_300_0000000000000000000000000000000000000000000000000000000000000002",
	}

	for _, f := range files {
		var fd *os.File
		filename := filepath.Join(dir, f)
		fd, err = os.Create(filename)
		if err != nil {
			panic(err)
		}
		_ = fd.Close()
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs := cache.Chunks()
	if len(cs) != 1 {
		t.Fatalf(fmt.Sprintf("chunks length should be 1 but get %d", len(cs)))
	} else {
		seg := cs[0]
		var zero types.Hash
		var one types.Hash
		one[len(one)-1] = 1
		if seg.Bound != [2]uint64{101, 200} {
			t.Fatal()
		} else if seg.PrevHash != zero {
			t.Fatal()
		} else if seg.Hash != one {
			t.Fatal()
		}
	}

	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func TestWriter(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = filepath.Join(dir, "mockw")
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		panic(err)
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	var seg = interfaces.Segment{
		Bound:    [2]uint64{101, 200},
		Hash:     types.Hash{},
		PrevHash: types.Hash{},
	}
	w, err := cache.NewWriter(seg)
	if err != nil {
		panic(err)
	}

	cs := cache.Chunks()
	if len(cs) != 0 {
		t.Fatalf(fmt.Sprintf("chunks length should be 0 but get %d", len(cs)))
	}

	err = w.Close()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to close writer: %v", err))
	}

	cs = cache.Chunks()
	if len(cs) != 1 {
		t.Fatalf(fmt.Sprintf("chunks length should be 1 but get %d", len(cs)))
	} else {
		seg = cs[0]
		var zero types.Hash
		if seg.Bound != [2]uint64{101, 200} {
			t.Fatal()
		} else if seg.PrevHash != zero {
			t.Fatal()
		} else if seg.Hash != zero {
			t.Fatal()
		}
	}

	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}
