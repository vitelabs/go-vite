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

func GetAbs(v int32) (int32, int32) {//abs, sign
	if v < 0 {
		return -v, -1
	} else {
		return v, 1
	}
}

func AdjustForDecimalsDiff(sourceAmountF *big.Float, decimalsDiff int32) *big.Float {
	if decimalsDiff == 0 {
		return sourceAmountF
	}
	dcDiffAbs, dcDiffSign := GetAbs(decimalsDiff)
	decimalDiffInt := new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(dcDiffAbs)), nil)
	decimalDiffFloat := new(big.Float).SetPrec(bigFloatPrec).SetInt(decimalDiffInt)
	if dcDiffSign > 0 {
		return sourceAmountF.Quo(sourceAmountF, decimalDiffFloat)
	} else {
		return sourceAmountF.Mul(sourceAmountF, decimalDiffFloat)
	}
}

func NegativeAmount(amount []byte) *big.Int {
	return new(big.Int).Neg(new(big.Int).SetBytes(amount))
}

func RoundAmount(amountF *big.Float) *big.Int {
	amount, _ := new(big.Float).SetPrec(bigFloatPrec).Add(amountF, big.NewFloat(0.5)).Int(nil)
	return amount
}
