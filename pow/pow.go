package pow

import (
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"

	"encoding/binary"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"golang.org/x/crypto/blake2b"
)

const (
	// todo this is online difficulty !!!
	// FullThreshold = 0xffffffc000000000
	FullThreshold = 0x000000000000FFFF
)

var DefaultDifficulty *big.Int = nil

func Init(vMTestParamEnabled bool) {
	if vMTestParamEnabled {
		DefaultDifficulty = new(big.Int).SetUint64(FullThreshold)
	}
}

// data = Hash(address + prehash); data + nonce < target.
func GetPowNonce(difficulty *big.Int, dataHash types.Hash) ([]byte, error) {
	data := dataHash.Bytes()
	for {
		nonce := crypto.GetEntropyCSPRNG(8)
		out := powHash256(nonce, data)
		if QuickGreater(out, helper.LeftPadBytes(difficulty.Bytes(), 32)) {
			return nonce, nil
		}
	}
	return nil, errors.New("get pow nonce error")
}

func powHash256(nonce []byte, data []byte) []byte {
	hash, _ := blake2b.New256(nil)
	hash.Write(nonce)
	hash.Write(data)
	out := hash.Sum(nil)
	return out
}

func CheckPowNonce(difficulty *big.Int, nonce []byte, data []byte) bool {
	out := powHash256(nonce, data)
	return QuickGreater(out, helper.LeftPadBytes(difficulty.Bytes(), 32))
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
	for i := 0; i < 32; i++ {
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
	return true
}

func Uint64ToByteArray(i uint64) [8]byte {
	var n [8]byte
	binary.LittleEndian.PutUint64(n[:], i)
	return n
}
