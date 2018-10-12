package pow_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/pow"
	"golang.org/x/crypto/blake2b"
	"math"
	"testing"
	"time"
)

func TestGetPowNonce(t *testing.T) {
	N := 20
	data := crypto.Hash256([]byte{1})
	timeList := make([]int64, N)
	for i := 0; i < N; i++ {
		startTime := time.Now()
		nonce := pow.GetPowNonce(nil, types.DataHash([]byte{1}))
		assert.True(t, pow.CheckPowNonce(nil, nonce, data))
		d := time.Now().Sub(startTime).Nanoseconds()
		fmt.Println("#", i, ":", d/1e6, "ms", "nonce", nonce)
		timeList[i] = d
	}

	max, min, timeSum, average, std := statistics(timeList)
	fmt.Println("average", average/1e6, "ms max", max/1e6, "ms min", min/1e6, "sum", timeSum/1e6, "standard deviation", std)
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

func TestQuickInc(t *testing.T) {
	data := []struct {
		x      []byte
		target []byte
		expect bool
	}{
		{[]byte{1, 2}, []byte{1, 3}, true},
		{[]byte{1, 0xFF}, []byte{2, 0}, true},
		{[]byte{0XFF, 0xFF}, []byte{0, 0}, true},
		{[]byte{0X1F, 0xFF}, []byte{0, 0}, false},
	}
	for _, v := range data {
		t1 := pow.QuickInc(v.x)
		fmt.Println(t1)
		assert.Equal(t, v.expect, bytes.Equal(t1, v.target))
	}
}

func TestHash256(t *testing.T) {
	hash2561 := blake2b.Sum256([]byte{1, 2, 1, 3})
	sum256 := crypto.Hash256([]byte{1, 2}, []byte{1, 3})
	assert.Equal(t, hash2561[:], sum256[:])
	assert.Equal(t, crypto.Hash256([]byte{1, 2}, []byte{1, 3}), sum256[:])
}
