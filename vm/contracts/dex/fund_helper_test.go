package dex

import (
	"bytes"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"strconv"
	"testing"
)

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
	assert.True(t, ValidPrice("10.5", true))
	assert.False(t, ValidPrice("00.000", true))
	assert.False(t, ValidPrice("-0.1", true))
	assert.False(t, ValidPrice("0..5", true))
	assert.True(t, ValidPrice("1.123456789012", true))
	assert.False(t, ValidPrice("1.1234567890123", true))
	assert.True(t, ValidPrice("123456789012.123456789", true))
	assert.False(t, ValidPrice("1234567890123.0", true))
	assert.True(t, ValidPrice("1234567890123", false))
	assert.False(t, ValidPrice("1234567890123", true))

	assert.True(t, ValidPrice(".24523", true))
	assert.False(t, ValidPrice("..24523", true))
	assert.False(t, ValidPrice("0.000", true))
	assert.False(t, ValidPrice("-.24523", true))
	assert.False(t, ValidPrice(".2452e3", true))
	assert.False(t, ValidPrice("3.2452e3", true))
}

func TestVerifyNewOrderPriceForRpc(t *testing.T) {
	data, err := cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundNewOrder, VxTokenId, ledger.ViteTokenId, true, uint8(Limited), "123456789012", big.NewInt(800))
	assert.Equal(t, nil, err)
	assert.True(t, VerifyNewOrderPriceForRpc(data))
	data, _ = cabi.ABIDexFund.PackMethod(cabi.MethodNameDexFundNewOrder, VxTokenId, ledger.ViteTokenId, true, uint8(Limited), "1234567890123", big.NewInt(800))
	assert.False(t, VerifyNewOrderPriceForRpc(data))
}

func TestPriceConvert(t *testing.T) {
	assert.Equal(t, "0.23", BytesToPrice(PriceToBytes(".23")))
	assert.Equal(t, "0.23", BytesToPrice(PriceToBytes("000.230000")))
	assert.Equal(t, "23", BytesToPrice(PriceToBytes("23.")))
	assert.Equal(t, "23", BytesToPrice(PriceToBytes("23.0")))

	assert.Equal(t, 0, bytes.Compare(PriceToBytes("23"), PriceToBytes("23.00")))
	assert.Equal(t, 1, bytes.Compare(PriceToBytes("23.001"), PriceToBytes("23")))
	assert.Equal(t, -1, bytes.Compare(PriceToBytes("23.001"), PriceToBytes("23.1")))
	assert.Equal(t, -1, bytes.Compare(PriceToBytes("0.021"), PriceToBytes("0.022")))
}

func TestRandomBytesFromBytes(t *testing.T) {
	data, err := hex.DecodeString("05672422d783f5b213836e276e865196f9677f78b366ca189c8a11453a67777e")
	assert.Equal(t, nil, err)
	assert.Equal(t, 32, len(data))
	codeBytes := []byte{'0', '0', '0', '0'}
	resMap := make(map[string]string)
	for i := 1; i < 3000; i++ {
		codeBytes, ok := randomBytesFromBytes(data, codeBytes, 0, 32)
		assert.True(t, ok)
		resStr := strconv.Itoa(int(BytesToUint32(codeBytes)))
		_, ok = resMap[resStr]
		if ok {
			break
		}
		assert.False(t, ok)
		resMap[resStr] = ""
		//fmt.Printf("%d -> %s\n", i, resStr)
	}
}
