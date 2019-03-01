package pow

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"golang.org/x/crypto/blake2b"
)

const (
	// todo this is online difficulty !!!
	// FullThreshold = 0xffffffc000000000
	FullThreshold = 0x000000000000FFFF
)

var defaultTarget = new(big.Int).SetUint64(FullThreshold)
var VMTestParamEnabled = false

func Init(vMTestParamEnabled bool) {
	VMTestParamEnabled = vMTestParamEnabled
}

// data = Hash(address + prehash); data + nonce < target.
func GetPowNonce(difficulty *big.Int, dataHash types.Hash) ([]byte, error) {
	var target *big.Int = nil
	if VMTestParamEnabled {
		target = defaultTarget
	} else {
		if difficulty == nil {
			return nil, errors.New("difficulty can't be nil")
		}
		target = DifficultyToTarget(difficulty)
		if target == nil || target.BitLen() > 256 {
			return nil, errors.New("target too long")
		}
	}

	data := dataHash.Bytes()
	target256 := helper.LeftPadBytes(target.Bytes(), 32)
	for {
		nonce := crypto.GetEntropyCSPRNG(8)
		out := powHash256(nonce, data)
		if QuickGreater(out, target256) {
			return nonce, nil
		}
	}
}

func powHash256(nonce []byte, data []byte) []byte {
	hash, _ := blake2b.New256(nil)
	hash.Write(nonce)
	hash.Write(data)
	out := hash.Sum(nil)
	return out
}

func CheckPowNonce(difficulty *big.Int, nonce []byte, data []byte) bool {
	var target *big.Int = nil
	if VMTestParamEnabled {
		target = defaultTarget
	} else {
		target = DifficultyToTarget(difficulty)
		if target == nil || target.BitLen() > 256 {
			return false
		}
	}
	out := powHash256(nonce, data)
	return QuickGreater(out, helper.LeftPadBytes(target.Bytes(), 32))
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

var (
	prec        uint = 64
	floatTwo256      = new(big.Float).SetPrec(prec).SetInt(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)))
	float1           = new(big.Float).SetPrec(prec).SetUint64(1)
)

func bigFloatToBigInt(f *big.Float) *big.Int {
	b, _ := new(big.Int).SetString(f.Text('f', 0), 10)
	return b
}

func DifficultyToTarget(difficulty *big.Int) *big.Int {
	fTmp := new(big.Float).SetPrec(prec).SetInt(difficulty)
	fTmp.Quo(float1, fTmp)
	fTmp.Add(fTmp, float1)
	fTmp.Quo(floatTwo256, fTmp)
	return bigFloatToBigInt(fTmp)
}
func TargetToDifficulty(target *big.Int) *big.Int {
	fTmp := new(big.Float).SetPrec(prec).SetInt(target)
	fTmp.Quo(floatTwo256, fTmp)
	fTmp.Sub(fTmp, float1)
	fTmp.Quo(float1, fTmp)
	return bigFloatToBigInt(fTmp)
}
