package govite

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func Benchmark_makeSlice(b *testing.B) {

	for i := 0; i < b.N; i++ {
		list := make([]ledger.AccountBlock, 0, 5)
		for i := 0; i < 10; i++ {
			a := ledger.AccountBlock{}
			list = append(list, a)

			b := list[i]
			list[i].Height = 10
			fmt.Println(b.Height)
			fmt.Println(list[i].Height)
			do(&b)
		}
	}
}

func Benchmark_makeSlice2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		list := make([]*ledger.AccountBlock, 0, 5)
		for i := 0; i < 10; i++ {
			a := ledger.AccountBlock{}
			list = append(list, &a)
			list[i].Height = 10
			b := list[i]
			fmt.Println(b.Height)
			fmt.Println(list[i].Height)
			do(b)
		}
	}
}

func do(b *ledger.AccountBlock) {

}
