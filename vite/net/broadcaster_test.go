package net

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/vite/net/circle"
)

func TestBroadcaster_Statistic(t *testing.T) {
	bdc := &broadcaster{
		statistic: circle.NewList(records24h),
	}

	ret := bdc.Statistic()
	for _, v := range ret {
		if v != 0 {
			t.Fail()
		}
	}

	// put one element
	fmt.Println("put one elements")
	bdc.statistic.Put(int64(10))
	ret = bdc.Statistic()
	if ret[0] != 10 {
		t.Fail()
	}

	// put 3600 elements
	fmt.Println("put 3610 elements")
	bdc.statistic.Reset()
	var t0 int64
	var t1 float64
	var total = records1h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
		t0 = int64(i)
	}
	fmt.Println(ret, t0, t1)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}

	// put 43200 elements
	fmt.Println("put 43210 elements")
	bdc.statistic.Reset()
	t0, t1 = 0, 0

	var t12 float64
	total = records12h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
	}
	for i := total - records12h; i < total; i++ {
		t12 += float64(i) / float64(records12h)
	}
	fmt.Println(ret, t0, t1, t12)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}
	if ret[2] != int64(t12) {
		t.Fail()
	}

	// put 86400 elements
	fmt.Println("put 86410 elements")
	bdc.statistic.Reset()
	t0, t1, t12 = 0, 0, 0

	var t24 float64
	total = records24h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
	}
	for i := total - records12h; i < total; i++ {
		t12 += float64(i) / float64(records12h)
	}
	for i := total - records24h; i < total; i++ {
		t24 += float64(i) / float64(records24h)
	}

	fmt.Println(ret, t0, t1, t12, t24)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}
	if ret[2] != int64(t12) {
		t.Fail()
	}
	if ret[3] != int64(t24) {
		t.Fail()
	}
}

func BenchmarkBroadcaster_Statistic(b *testing.B) {
	bdc := &broadcaster{
		statistic: circle.NewList(records24h),
	}

	for i := int64(0); i < records24h*2; i++ {
		bdc.statistic.Put(i)
	}

	var ret []int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret = bdc.Statistic()
	}
	fmt.Println(ret)
}

func TestMemStore_EnqueueAccountBlock(t *testing.T) {
	store := newMemBlockStore(1000)
	for i := uint64(0); i < 1100; i++ {
		store.enqueueAccountBlock(&ledger.AccountBlock{
			Height: i,
		})
	}
	for i := uint64(0); i < 1000; i++ {
		block := store.dequeueAccountBlock()
		if block.Height != i {
			t.Fail()
		}
	}
	for i := uint64(0); i < 100; i++ {
		block := store.dequeueAccountBlock()
		if block != nil {
			t.Fail()
		}
	}
}

func TestMemStore_EnqueueSnapshotBlock(t *testing.T) {
	store := newMemBlockStore(1000)
	for i := uint64(0); i < 1100; i++ {
		store.enqueueSnapshotBlock(&ledger.SnapshotBlock{
			Height: i,
		})
	}
	for i := uint64(0); i < 1000; i++ {
		block := store.dequeueSnapshotBlock()
		if block.Height != i {
			t.Fail()
		}
	}
	for i := uint64(0); i < 100; i++ {
		block := store.dequeueSnapshotBlock()
		if block != nil {
			t.Fail()
		}
	}
}
