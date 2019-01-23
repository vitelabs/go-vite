package dex

import (
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
)

func AddBigInt(a []byte, b []byte) []byte {
	return new(big.Int).Add(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b)).Bytes()
}

func SubBigIntAbs(a []byte, b []byte) []byte {
	return new(big.Int).Sub(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b)).Bytes()
}

func SubBigInt(a []byte, b []byte) *big.Int {
	return new(big.Int).Sub(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b))
}

func MinBigInt(a []byte, b []byte) []byte {
	if new(big.Int).SetBytes(a).Cmp(new(big.Int).SetBytes(b)) > 0 {
		return b
	} else {
		return a
	}
}

func CmpToBigZero(a []byte) int {
	return new(big.Int).SetBytes(a).Sign()
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

func SubForAbsAndSign(a, b int32) (int32, int32) {
	r := a - b
	if r < 0 {
		return -r, -1
	} else {
		return r, 1
	}
}

func AdjustForDecimalsDiff(sourceAmountF *big.Float, sourceDecimals, targetDecimals int32) *big.Float {
	if sourceDecimals == targetDecimals {
		return sourceAmountF
	}
	dcDiffAbs, dcDiffSign := SubForAbsAndSign(sourceDecimals, targetDecimals)
	decimalDiffInt := new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(dcDiffAbs)), nil)
	decimalDiffFloat := new(big.Float).SetInt(decimalDiffInt)
	if dcDiffSign > 0 {
		return sourceAmountF.Quo(sourceAmountF, decimalDiffFloat)
	} else {
		return sourceAmountF.Mul(sourceAmountF, decimalDiffFloat)
	}
}

func NegativeAmount(amount []byte) *big.Int {
	return new(big.Int).Sub(big.NewInt(0), new(big.Int).SetBytes(amount))
}