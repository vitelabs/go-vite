package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/vite/net/circle"
	"testing"
)

func TestBroadcaster_Statistic(t *testing.T) {
	bdc := &broadcaster{
		statis: circle.NewList(records_24),
	}

	ret := bdc.Statistic()
	for _, v := range ret {
		if v != 0 {
			t.Fail()
		}
	}

	// put one element
	fmt.Println("put one elements")
	bdc.statis.Put(int64(10))
	ret = bdc.Statistic()
	if ret[0] != 10 {
		t.Fail()
	}

	// put 3600 elements
	fmt.Println("put 3610 elements")
	bdc.statis.Reset()
	var t0 int64
	var t1 float64
	var total = records_1 + 10
	for i := 0; i < total; i++ {
		bdc.statis.Put(int64(i))
	}
	ret = bdc.Statistic()
	for i := total - records_1; i < total; i++ {
		t1 += float64(i) / float64(records_1)
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
	bdc.statis.Reset()
	t0, t1 = 0, 0

	var t12 float64
	total = records_12 + 10
	for i := 0; i < total; i++ {
		bdc.statis.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records_1; i < total; i++ {
		t1 += float64(i) / float64(records_1)
	}
	for i := total - records_12; i < total; i++ {
		t12 += float64(i) / float64(records_12)
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
	bdc.statis.Reset()
	t0, t1, t12 = 0, 0, 0

	var t24 float64
	total = records_24 + 10
	for i := 0; i < total; i++ {
		bdc.statis.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records_1; i < total; i++ {
		t1 += float64(i) / float64(records_1)
	}
	for i := total - records_12; i < total; i++ {
		t12 += float64(i) / float64(records_12)
	}
	for i := total - records_24; i < total; i++ {
		t24 += float64(i) / float64(records_24)
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
		statis: circle.NewList(records_24),
	}

	for i := int64(0); i < records_24*2; i++ {
		bdc.statis.Put(i)
	}

	var ret []int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret = bdc.Statistic()
	}
	fmt.Println(ret)
}
