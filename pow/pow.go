package pow

import (
	"bytes"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

// IN MY 2017 MACBOOK PRO which cpu is---- Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz----, that target costs about 2.14 seconds
// average 2.1429137205785e+09 max 35658900929 min 63118 sum 21429137205785 standard deviation 2.381750598860289e+09
var DummyTarget, _ = new(big.Int).SetString("000003FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)

// data = Hash(address + prehash); nonce + data < target. if prehash == nil {data = Hash(address)}
func GetPowNonce(target *big.Int, data []byte) *big.Int {
	if target == nil {
		return nil
	}

	nonce := crypto.GetEntropyCSPRNG(32)
	targetBytes := target.Bytes()
	for {
		if QuickLess(crypto.Hash256(nonce, data), targetBytes) {
			break
		}
		nonce = QuickInc(nonce)
	}
	return new(big.Int).SetBytes(nonce)
}

func CheckNonce(target, nonce *big.Int, data []byte) bool {
	if target == nil || nonce == nil {
		return false
	}
	return new(big.Int).SetBytes(crypto.Hash256(nonce.Bytes(), data)).Cmp(target) < 0
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
func QuickLess(x, y []byte) bool {
	return bytes.Compare(x, y) < 0
}
