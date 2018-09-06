package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

const (
	maxUint64 = 1<<64 - 1

	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8

	retry   = true
	noRetry = false
)

var (
	Big0   = big.NewInt(0)
	Big1   = big.NewInt(1)
	Big32  = big.NewInt(32)
	big256 = big.NewInt(256)
	big257 = big.NewInt(257)

	bigZero = new(big.Int)

	tt255   = BigPow(2, 255)
	tt256   = BigPow(2, 256)
	tt256m1 = new(big.Int).Sub(tt256, big.NewInt(1))

	emptyHash        = types.Hash{}
	emptyAddress     = types.Address{}
	emptyTokenTypeId = types.TokenTypeId{}
	emptyWord        = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// TODO system id
	viteTokenTypeId = types.TokenTypeId{}
	snapshotGid     = Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
)

func isViteToken(tokenId types.TokenTypeId) bool {
	return bytes.Equal(tokenId.Bytes(), viteTokenTypeId.Bytes())
}
func isSnapshotGid(gid Gid) bool {
	return bytes.Equal(gid.Bytes(), snapshotGid.Bytes())
}

// toWordSize returns the ceiled word size required for memory expansion.
func toWordSize(size uint64) uint64 {
	if size > maxUint64-31 {
		return maxUint64/32 + 1
	}

	return (size + 31) / 32
}

// bigUint64 returns the integer casted to a uint64 and returns whether it
// overflowed in the process.
func bigUint64(v *big.Int) (uint64, bool) {
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

// getDataBig returns a slice from the data based on the start and size and pads
// up to size with zero's. This function is overflow safe.
func getDataBig(data []byte, start *big.Int, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := BigMin(start, dlen)
	e := BigMin(new(big.Int).Add(s, size), dlen)
	return RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

func useQuota(quota, cost uint64) (uint64, error) {
	if quota < cost {
		return 0, ErrOutOfQuota
	}
	quota = quota - cost
	return quota, nil
}
func useQuotaForData(data []byte, quota uint64) (uint64, error) {
	cost, err := dataGasCost(data)
	if err != nil {
		return 0, err
	}
	return useQuota(quota, cost)
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

func allZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}

func joinBytes(data ...[]byte) []byte {
	newData := []byte{}
	for _, d := range data {
		newData = append(newData, d...)
	}
	return newData
}
