package pow

import (
	"github.com/vitelabs/go-vite/crypto"
	"golang.org/x/crypto/blake2b"
	"math/big"
)

// IN MY 2017 MACBOOK PRO which cpu is---- Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz----, that target costs about 1.8 seconds
// average 2.1429137205785e+09 max 35658900929 min 63118 sum 21429137205785 standard deviation 2.381750598860289e+09
var DummyTarget, _ = new(big.Int).SetString("000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)

const size = 32

// data = Hash(address + prehash); data + nonce < target. if prehash == nil {data = Hash(address)}
func GetPowNonce(target *big.Int, data []byte) *big.Int {
	if target == nil {
		return nil
	}
	nonce := crypto.GetEntropyCSPRNG(size)
	targetBytes := Fixed32SizeBytes(target.Bytes())
	calcBytes := make([]byte, 64)
	l := copy(calcBytes, data)
	copy(calcBytes[l:], nonce)
	for {
		if QuickLess(blake2b.Sum256(calcBytes), targetBytes) {
			break
		}
		calcBytes = QuickInc(calcBytes)
	}
	return new(big.Int).SetBytes(calcBytes[size:])
}

func CheckNonce(target, nonce *big.Int, data []byte) bool {
	if target == nil || nonce == nil {
		return false
	}
	v := crypto.Hash256(data, nonce.Bytes())
	return new(big.Int).SetBytes(v).Cmp(target) < 0
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

func QuickLess(x, y [size]byte) bool {
	for i := 0; i < size; i++ {
		if x[i] > y[i] {
			return false
		}
		if x[i] < y[i] {
			return true
		}
		if x[i] == y[i] {
			continue
		}
	}
	return false
}

func Fixed32SizeBytes(b []byte) [size]byte {
	r := [size]byte{}
	delta := size - len(b)
	if delta > 0 {
		copy(r[delta:], b)
	}
	return r
}
