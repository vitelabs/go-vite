package ledger

import (
	"fmt"
	"testing"
)

type Bclass struct {
	HAHA uint64
}

type Aclass struct {
	b  Bclass
	Ts []uint64
}

func TestAccountBlock_Copy(t *testing.T) {
	a := Aclass{
		b: Bclass{
			HAHA: 12,
		},
		Ts: []uint64{1, 2, 3},
	}
	fmt.Println(a.Ts)

	d := a
	fmt.Println(d.Ts)

	d.Ts[0] = 10
	fmt.Println(d.Ts)

	fmt.Println(a.Ts)

}
