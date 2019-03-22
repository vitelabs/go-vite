package helper

import (
	"encoding/hex"
	"math/big"
)

const (
	MaxUint64 = uint64(1<<64 - 1)

	// number of bits in chain big.Word
	WordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in chain big.Word
	WordBytes = WordBits / 8
	// number of bytes in chain vm word
	WordSize = 32
)

var (
	Big0   = big.NewInt(0)
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big10  = big.NewInt(10)
	Big31  = big.NewInt(31)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)

	Tt255   = BigPow(2, 255)
	Tt256   = BigPow(2, 256)
	Tt256m1 = new(big.Int).Sub(Tt256, big.NewInt(1))
)

// ToWordSize returns the ceiled word size required for memory expansion.
func ToWordSize(size uint64) uint64 {
	if size > MaxUint64-WordSize+1 {
		return MaxUint64/WordSize + 1
	}

	return (size + WordSize - 1) / WordSize
}

// BigUint64 returns the integer casted to chain uint64 and returns whether it
// overflowed in the process.
func BigUint64(v *big.Int) (uint64, bool) {
	return v.Uint64(), v.BitLen() > 64
}

// rightPadBytes zero-pads slice to the right up to length l.
func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}

// leftPadBytes zero-pads slice to the left up to length l.
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

// GetDataBig returns chain slice from the data based on the start and size and pads
// up to size with zero's. This function is overflow safe.
func GetDataBig(data []byte, start *big.Int, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := BigMin(start, dlen)
	e := BigMin(new(big.Int).Add(s, size), dlen)
	return RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

func BytesToString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func HexToBytes(str string) []byte {
	data, _ := hex.DecodeString(str)
	return data
}

func AllZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}

func JoinBytes(data ...[]byte) []byte {
	newData := []byte{}
	for _, d := range data {
		newData = append(newData, d...)
	}
	return newData
}
