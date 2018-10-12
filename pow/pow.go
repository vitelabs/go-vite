package pow

import (
	"encoding/binary"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

// IN MY 2017 MACBOOK PRO which cpu is---- Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz----
// average 2.17099039203e+10 max 73782690184 min 641170149 sum 217099039203 standard deviation 2.0826136795592163e+10
const (
	// todo this is online difficulty !!!
	// FullThreshold = 0xffffffc000000000
	FullThreshold = 0x000000000000FFFF
)

// data = Hash(address + prehash); data + nonce < target.
func GetPowNonce(difficulty *big.Int, dataHash types.Hash) [8]byte {
	rng := crypto.GetEntropyCSPRNG(8)
	data := dataHash.Bytes()

	calc, target := prepareData(difficulty, data, rng)
	for {
		if QuickGreater(crypto.Hash(8, calc), target[:]) {
			break
		}
		calc = QuickInc(calc)
	}

	var arr [8]byte
	copy(arr[:], calc[32:])
	return arr
}

func CheckPowNonce(difficulty *big.Int, nonce [8]byte, data []byte) bool {
	calc, target := prepareData(difficulty, data, nonce[:])
	return QuickGreater(crypto.Hash(8, calc), target[:])
}

func prepareData(difficulty *big.Int, data []byte, nonce []byte) ([]byte, [8]byte) {
	threshold := getThresholdByDifficulty(difficulty)
	calc := make([]byte, 40)
	l := copy(calc, data)
	copy(calc[l:], nonce[:])
	target := Uint64ToByteArray(threshold)
	return calc, target
}

func getThresholdByDifficulty(difficulty *big.Int) uint64 {
	if difficulty != nil {
		return difficulty.Uint64()
	} else {
		return FullThreshold
	}
}

func QuickInc(x []byte) []byte {
	for i := 1; i <= len(x); i++ {
		x[len(x)-i] = x[len(x)-i] + 1
		if x[len(x)-i] != 0 {
			return x
		}
	}
	return x
}

func QuickGreater(x, y []byte) bool {
	for i := 0; i < 8; i++ {
		if x[i] > y[i] {
			return true
		}
		if x[i] < y[i] {
			return false
		}
		if x[i] == y[i] {
			continue
		}
	}
	return false
}

func Uint64ToByteArray(i uint64) [8]byte {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], i)
	return n
}
