package compress

import (
	"testing"
)

func BenchmarkFile1(b *testing.B) {
	b.StartTimer()
	for i := 0; i < 10000000; i++ {
		a := [64]byte{0, 4, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
		testPointer(&a)
	}
	b.StopTimer()
}

func BenchmarkFile2(b *testing.B) {
	b.StartTimer()
	for i := 0; i < 10000000; i++ {
		a := [64]byte{1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
		testArray(a)
	}
	b.StopTimer()
}

func testArray(a [64]byte) [64]byte {
	d := a
	return d
}

func testPointer(a *[64]byte) *[64]byte {
	d := a
	return d
}
