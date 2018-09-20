package pow_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/pow"
	"math"
	"math/big"
	"testing"
	"time"
)

func TestGetPowNonce(t *testing.T) {
	N := 10000
	data := crypto.Hash256([]byte{1})
	timeList := make([]int64, N)
	for i := 0; i < N; i++ {
		startTime := time.Now()
		nonce := pow.GetPowNonce(pow.DummyTarget, data)
		assert.True(t, pow.CheckNonce(pow.DummyTarget, nonce, data))
		d := time.Now().Sub(startTime).Nanoseconds()
		fmt.Println("#", i, ":", d/1e6, "ms", "nonce", nonce)
		timeList[i] = d
	}

	d, _ := new(big.Int).SetString("1", 10)
	assert.False(t, pow.CheckNonce(pow.DummyTarget, d, data))
	max, min, timeSum, average, std := statistics(timeList)
	fmt.Println("average", average, "max", max, "min", min, "sum", timeSum, "standard deviation", std)
}

func statistics(data []int64) (timeMax, timeMin, timeSum int64, average, std float64) {
	timeSum = 0
	timeMax = 0
	timeMin = 1 << 31
	average = 0
	std = 0
	for _, v := range data {
		timeSum += v
		if v > timeMax {
			timeMax = v
		}
		if v < timeMin {
			timeMin = v
		}
	}
	average = float64(timeSum) / float64(len(data))
	vSum := float64(0)
	for _, v := range data {
		vSum += (float64(v) - average) * (float64(v) - average)
	}
	std = math.Sqrt(vSum / float64(len(data)))
	return timeMax, timeMin, timeSum, average, std
}
