package core

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type VoteType uint8

const (
	NORMAL                 = VoteType(0)
	SUCCESS_RATE_PROMOTION = VoteType(1)
	SUCCESS_RATE_DEMOTION  = VoteType(2)
	RANDOM_PROMOTION       = VoteType(3)
)

type Vote struct {
	Name    string
	Addr    types.Address
	Balance *big.Int
	Type    []VoteType
}

type ByBalance []*Vote

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[j].Balance.Cmp(a[i].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	} else {
		return r < 0
	}
}
