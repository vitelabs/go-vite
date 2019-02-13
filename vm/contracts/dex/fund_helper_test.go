package dex

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestGetMindedVxAmt(t *testing.T) {
	_, _, _, result := GetMindedVxAmt(big.NewInt(0))
	assert.False(t, result)

	balance, _ := new(big.Float).Mul(new(big.Float).SetFloat64(1.5), new(big.Float).SetInt(VxMinedAmtPerPeriod)).Int(nil)
	amtFroFeePerMarket, amtForPledge, amtForViteLabs, result :=GetMindedVxAmt(balance)
	assert.True(t, result)
	assert.True(t, amtFroFeePerMarket.Cmp(new(big.Int).Mul(vxTokenPow, big.NewInt(27400))) == 0)
	assert.True(t, amtForPledge.Cmp(new(big.Int).Mul(vxTokenPow, big.NewInt(1370))) == 0)
	assert.True(t, amtForViteLabs.Cmp(new(big.Int).Mul(vxTokenPow, big.NewInt(1370))) == 0)

	balance1 := big.NewInt(13)
	amtFroFeePerMarket, amtForPledge, amtForViteLabs, result = GetMindedVxAmt(balance1)
	assert.True(t, result)
	assert.True(t, amtFroFeePerMarket.Cmp(big.NewInt(3)) == 0)
	assert.True(t, amtForViteLabs.Cmp(big.NewInt(1)) == 0)
	assert.True(t, amtForPledge.Cmp(big.NewInt(0)) == 0)
}

func TestDivideByProportion(t *testing.T) {
	totalReferAmt := big.NewInt(133)
	partReferAmt := big.NewInt(13)
	dividedReferAmt := big.NewInt(11)
	toDivideTotalAmt := big.NewInt(127)
	toDivideLeaveAmt := big.NewInt(15)

	proportionAmt, finished := DivideByProportion(totalReferAmt, partReferAmt, dividedReferAmt, toDivideTotalAmt, toDivideLeaveAmt)
	assert.False(t, finished)
	assert.True(t, proportionAmt.Int64() == 12)
	assert.True(t, dividedReferAmt.Int64() == 24)
	assert.True(t, toDivideLeaveAmt.Int64() == 3)

	partReferAmt = big.NewInt(10)
	proportionAmt, finished = DivideByProportion(totalReferAmt, partReferAmt, dividedReferAmt, toDivideTotalAmt, toDivideLeaveAmt)
	assert.True(t, finished)
	assert.True(t, proportionAmt.Int64() == 3)
	assert.True(t, dividedReferAmt.Int64() == 34)

	toDivideLeaveAmt = big.NewInt(15)
	dividedReferAmt = big.NewInt(130)
	partReferAmt = big.NewInt(5)
	proportionAmt, finished = DivideByProportion(totalReferAmt, partReferAmt, dividedReferAmt, toDivideTotalAmt, toDivideLeaveAmt)
	assert.True(t, finished)
	assert.True(t, proportionAmt.Int64() == 15)
}

func TestValidPrice(t *testing.T) {
	assert.True(t, ValidPrice("10.5"))
	assert.False(t, ValidPrice("0.0"))
	assert.False(t, ValidPrice("-0.1"))
	assert.False(t, ValidPrice("0..5"))
	assert.False(t, ValidPrice("1.123456789"))
	assert.True(t, ValidPrice("1.12345678"))
}

