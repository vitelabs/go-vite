package pow

import (
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

// IN MY 2017 MACBOOK PRO which cpu is---- Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz----, that target costs about 2 seconds
const DUMMY_TARGET = "000003FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

// data = prehash + address; nonce + data < target
func GetPowNonce(target *big.Int, data []byte) *big.Int {
	if target == nil {
		target, _ = new(big.Int).SetString(DUMMY_TARGET, 16)
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
