package api

import (
	"testing"
	"math/big"
	"github.com/vitelabs/go-vite/common/math"
	"math/rand"
	"fmt"
)

func TestTestApi_GetTestToken(t *testing.T) {
	a := rand.Int() % 1000
	a += 1
	ba := new(big.Int).SetInt64(int64(a))
	ba.Mul(ba, math.BigPow(10, 18))

	amount := ba.String()
	fmt.Println(amount)
}
