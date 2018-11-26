package dex

import "math/big"

func AddBigInt(a []byte, b []byte) []byte {
	return new(big.Int).Add(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b)).Bytes()
}


func SubBigInt(a []byte, b []byte) []byte {
	return new(big.Int).Sub(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b)).Bytes()
}

func MinBigInt(a []byte, b []byte) []byte {
	if new(big.Int).SetBytes(a).Cmp(new(big.Int).SetBytes(b)) > 0 {
		return b
	} else {
		return a
	}
}

func CmpToBigZero(a []byte) int {
	if len(a) == 0 {
		return 0
	}
	return new(big.Int).SetBytes(a).Cmp(big.NewInt(0))
}

func CmpForBigInt(a []byte, b []byte) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	} else if len(a) == 0 {
		return -1
	} else if len(b) == 0 {
		return 1
	}
	return new(big.Int).SetBytes(a).Cmp(new(big.Int).SetBytes(b))
}