package sync_cache

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func TestSyncLoad3(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = path.Join(dir, "sync_cache")
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs2 := interfaces.SegmentList{
		{
			From:     1,
			To:       100,
			PrevHash: types.Hash{0},
			Hash:     types.Hash{1},
		},
		{
			From:     101,
			To:       200,
			PrevHash: types.Hash{1},
			Hash:     types.Hash{2},
		},
		{
			From:     301,
			To:       400,
			PrevHash: types.Hash{3},
			Hash:     types.Hash{4},
		},
	}

	for _, c := range cs2 {
		if _, err = cache.NewWriter(c, 0); err != nil {
			panic(err)
		}
	}

	err = cache.Close()
	if err != nil {
		panic(err)
	}

	cache, err = NewSyncCache(dir)
	cs := cache.Chunks()
	if len(cs) > 0 {
		t.Error("should not cache")
	}
}

func TestSyncLoad2(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = path.Join(dir, "sync_cache")
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs2 := interfaces.SegmentList{
		{
			From:     1,
			To:       100,
			PrevHash: types.Hash{0},
			Hash:     types.Hash{1},
		},
		{
			From:     101,
			To:       200,
			PrevHash: types.Hash{1},
			Hash:     types.Hash{2},
		},
		{
			From:     301,
			To:       400,
			PrevHash: types.Hash{3},
			Hash:     types.Hash{4},
		},
	}

	for _, c := range cs2 {
		if w, err := cache.NewWriter(c, 0); err != nil {
			panic(err)
		} else {
			err = w.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	err = cache.Close()
	if err != nil {
		panic(err)
	}

	cache, err = NewSyncCache(dir)
	cs := cache.Chunks()
	if len(cs) != len(cs2) {
		t.Error("different chunks")
	}
	for i, c := range cs {
		if !cs2[i].Equal(c) {
			t.Error("different chunk")
		}
	}
}

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

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs2 := interfaces.SegmentList{
		{
			From:     2,
			To:       100,
			PrevHash: types.Hash{1},
			Hash:     types.Hash{2},
		},
		{
			From:     101,
			To:       200,
			PrevHash: types.Hash{3},
			Hash:     types.Hash{4},
		},
		{
			From:     301,
			To:       400,
			PrevHash: types.Hash{7},
			Hash:     types.Hash{8},
		},
	}

	for _, c := range cs2 {
		if w, err := cache.NewWriter(c, 0); err != nil {
			panic(err)
		} else {
			err = w.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	cs := cache.Chunks()
	if len(cs) == len(cs2) {
		for i, c := range cs2 {
			if !cs[i].Equal(c) {
				t.Error("different chunk")
			}
		}
	} else {
		t.Error("different chunks")
	}

	w, err := cache.NewWriter(interfaces.Segment{
		From: 90,
		To:   190,
	}, 0)
	if err == nil {
		panic("should not create writer")
	}

	reader, err := cache.NewReader(cs2[0])
	if err != nil {
		panic(err)
	}
	if reader.Verified() {
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
	if reader.Verified() {
		t.Error("should not verified")
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
		From:     401,
		To:       500,
		Hash:     types.Hash{9},
		PrevHash: types.Hash{10},
	}
	w, err = cache.NewWriter(seg, 0)
	if err != nil {
		panic(err)
	}
	cs = cache.Chunks()
	for _, c := range cs {
		if c.Equal(seg) {
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
		if c.Equal(seg) {
			in = true
		}
	}
	if false == in {
		t.Error("write close, should in chunk list")
	}
}

func TestSyncCache_Delete(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = path.Join(dir, "sync_cache")
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	cs2 := interfaces.SegmentList{
		{
			From:     2,
			To:       100,
			PrevHash: types.Hash{1},
			Hash:     types.Hash{2},
		},
		{
			From:     101,
			To:       200,
			PrevHash: types.Hash{3},
			Hash:     types.Hash{4},
		},
		{
			From:     301,
			To:       400,
			PrevHash: types.Hash{7},
			Hash:     types.Hash{8},
		},
	}

	for _, c := range cs2 {
		if w, err := cache.NewWriter(c, 0); err != nil {
			panic(err)
		} else {
			err = w.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	err = cache.Delete(interfaces.Segment{
		From:     2,
		To:       100,
		PrevHash: types.Hash{1},
		Hash:     types.Hash{2},
	})
	if err != nil {
		panic(err)
	}

	cs := cache.Chunks()
	if len(cs) == len(cs2)-1 {
		for i := 1; i < len(cs2); i++ {
			if !cs2[i].Equal(cs[i-1]) {
				t.Error("different chunk")
			}
		}
	} else {
		t.Error("different chunks")
	}
}

func TestSyncCache_NewWriter(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	dir = path.Join(dir, "sync_cache")
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	cache, err := NewSyncCache(dir)
	if err != nil {
		panic(err)
	}

	seg := interfaces.Segment{
		From:     1,
		To:       100,
		Hash:     types.Hash{1},
		PrevHash: types.Hash{100},
	}
	w, err := cache.NewWriter(seg, 1000)
	if err != nil {
		panic(err)
	}

	err = w.Close()
	if err != nil {
		panic(err)
	}

	var find bool
	cs := cache.Chunks()
	for _, c := range cs {
		if c.Equal(seg) {
			find = true
		}
	}

	if !find {
		t.Fail()
	}
}
