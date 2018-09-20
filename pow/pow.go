package pow

import (
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

// IN MY 2017 MACBOOK PRO which cpu is---- Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz----, that target costs about 2 seconds
var DummyTarget, _ = new(big.Int).SetString("000003FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)

// data = Hash(address + prehash); nonce + data < target. if prehash == nil {data = Hash(address)}
func GetPowNonce(target *big.Int, data []byte) *big.Int {
	if target == nil {
		return nil
	}

	csprng := crypto.GetEntropyCSPRNG(32)
	from := new(big.Int).SetBytes(csprng)
	calc := new(big.Int)
	step := big.NewInt(1)
	for {
		calc.SetBytes(crypto.Hash256(from.Bytes(), data))
		if calc.Cmp(target) < 0 {
			break
		}
		from = from.Add(from, step)
	}
	return from
}

func CheckNonce(target, nonce *big.Int, data []byte) bool {
	if target == nil || nonce == nil {
		return false
	}
	return new(big.Int).SetBytes(crypto.Hash256(nonce.Bytes(), data)).Cmp(target) < 0
}
