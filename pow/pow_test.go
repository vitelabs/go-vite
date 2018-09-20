package pow_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/pow"
	"testing"
	"time"
)

func TestPow(t *testing.T) {
	N := 1000
	data := crypto.Hash256([]byte{1})
	timesum := int64(0)
	timeMax := int64(0)
	timeMin := int64(1 << 31)
	for i := 0; i < N; i++ {
		startTime := time.Now()
		nonce := pow.GetPowNonce(nil, data)

		d := time.Now().Sub(startTime).Nanoseconds()
		fmt.Println("#", i, ":", d/1e6, "ms", "nonce", nonce)
		timesum += d
		if d > timeMax {
			timeMax = d
		}
		if d < timeMin {
			timeMin = d
		}
	}

	println("average", timesum/(int64(N))/1e6, "ms", "max", timeMax, "ns", "min", timeMin, "ns")
}
