package dex

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFeeAndExecutedFee(t *testing.T) {
	innerTestCalculateFeeAndExecutedFee(t, false)
	innerTestCalculateFeeAndExecutedFee(t, true)
}

func innerTestCalculateFeeAndExecutedFee(t *testing.T, isDexFeeFork bool) {
	maker := &Order{}
	maker.Side = false
	maker.Quantity = big.NewInt(1000000000).Bytes()
	maker.Amount = big.NewInt(10000).Bytes()
	maker.LockedBuyFee = big.NewInt(25).Bytes()
	maker.Price = PriceToBytes("0.00001")
	maker.MakerFeeRate = BaseFeeRate
	maker.MakerOperatorFeeRate = 50

	executeQuantity := big.NewInt(988500000).Bytes()
	executeAmount := big.NewInt(9885).Bytes()
	_, makerExecutedFee, _, makerExecutedOperatorFee := CalculateFeeAndExecutedFee(maker, executeAmount, maker.MakerFeeRate, maker.MakerOperatorFeeRate, isDexFeeFork)
	updateOrder(maker, executeQuantity, executeAmount, makerExecutedFee, makerExecutedOperatorFee, 0)
	assert.Equal(t, "5", new(big.Int).SetBytes(maker.ExecutedOperatorFee).String())

	executeQuantity = big.NewInt(11500000).Bytes()
	executeAmount = big.NewInt(115).Bytes()
	_, makerExecutedFee, _, makerExecutedOperatorFee = CalculateFeeAndExecutedFee(maker, executeAmount, maker.MakerFeeRate, maker.MakerOperatorFeeRate, isDexFeeFork)
	updateOrder(maker, executeQuantity, executeAmount, makerExecutedFee, makerExecutedOperatorFee, 0)
	if isDexFeeFork {
		assert.Equal(t, "5", new(big.Int).SetBytes(maker.ExecutedOperatorFee).String())
	} else {
		assert.Equal(t, "0", new(big.Int).SetBytes(maker.ExecutedOperatorFee).String())
	}
}

func TestSafeSubBigInt(t *testing.T) {
	amt := big.NewInt(10).Bytes()
	sub := big.NewInt(9).Bytes()
	res, actualSub, exceed := SafeSubBigInt(amt, sub)
	assert.Equal(t, "1", new(big.Int).SetBytes(res).String())
	assert.Equal(t, "9", new(big.Int).SetBytes(actualSub).String())
	assert.False(t, exceed)

	amt = big.NewInt(10).Bytes()
	sub = big.NewInt(10).Bytes()
	res, actualSub, exceed = SafeSubBigInt(amt, sub)
	assert.True(t, len(res) == 0)
	assert.True(t, bytes.Equal(actualSub, amt))
	assert.False(t, exceed)

	amt = big.NewInt(10).Bytes()
	sub = big.NewInt(11).Bytes()
	res, actualSub, exceed = SafeSubBigInt(amt, sub)
	assert.True(t, len(res) == 0)
	assert.True(t, bytes.Equal(actualSub, amt))
	assert.True(t, exceed)

	amt = big.NewInt(0).Bytes()
	sub = big.NewInt(11).Bytes()
	res, actualSub, exceed = SafeSubBigInt(amt, sub)
	assert.True(t, len(res) == 0)
	assert.True(t, bytes.Equal(actualSub, amt))
	assert.True(t, exceed)
}