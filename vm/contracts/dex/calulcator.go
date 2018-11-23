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