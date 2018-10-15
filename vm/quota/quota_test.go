package quota

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestCalcQuotaUsed(t *testing.T) {
	tests := []struct {
		quotaTotal, quotaAddition, quotaLeft, quotaRefund, quotaUsed uint64
		err                                                          error
	}{
		{15000, 5000, 10001, 0, 0, nil},
		{15000, 5000, 9999, 0, 1, nil},
		{10000, 0, 9999, 0, 1, nil},
		{10000, 0, 5000, 1000, 4000, nil},
		{10000, 0, 5000, 5000, 2500, nil},
		{15000, 5000, 5000, 5000, 2500, nil},
		{15000, 5000, 10001, 0, 10000, ErrOutOfQuota},
		{15000, 5000, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 1000, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 10001, 0, 0, errors.New("")},
		{15000, 5000, 9999, 0, 1, errors.New("")},
		{10000, 0, 9999, 0, 1, errors.New("")},
		{10000, 0, 5000, 1000, 5000, errors.New("")},
		{15000, 5000, 5000, 5000, 5000, errors.New("")},
	}
	for i, test := range tests {
		quotaUsed := CalcQuotaUsed(test.quotaTotal, test.quotaAddition, test.quotaLeft, test.quotaRefund, test.err)
		if quotaUsed != test.quotaUsed {
			t.Fatalf("%v th calculate quota used failed, expected %v, got %v", i, test.quotaUsed, quotaUsed)
		}
	}
}

func TestUseQuota(t *testing.T) {
	tests := []struct {
		quotaInit, cost, quotaLeft uint64
		err                        error
	}{
		{100, 100, 0, nil},
		{100, 101, 0, ErrOutOfQuota},
	}
	for _, test := range tests {
		quotaLeft, err := UseQuota(test.quotaInit, test.cost)
		if quotaLeft != test.quotaLeft || err != test.err {
			t.Fatalf("use quota fail, input: %v, %v, expected [%v, %v], got [%v, %v]", test.quotaInit, test.cost, test.quotaLeft, test.err, quotaLeft, err)
		}
	}
}

// quota = Qm * (1 - 2 / (1 + e**(paramPledge * Hgap * pledgeAmount + paramPoW * difficulty)))
// Qm is decided by total quota used during past 3600 snapshot blocks,
// using 1 million for the sake of simplicity in test net.
// Default difficulty is 0xffffffc000000000.
func TestCalcLogisticQuotaParam(t *testing.T) {
	quotaLimit := 1000000.0
	quotaForPureTransaction := 21000.0

	// Pledge minimum amount of Vite Token, calc no PoW, wait for longest block height, gets quota for a pure transfer transaction
	// maxHeightGap := 86400.0
	// Pledge minimum amount of Vite Token, calc no PoW, wait for one snapshot block, gets quota for a pure transfer transaction
	maxHeightGap := 1.0
	minPledgeAmount := 1.0e19
	paramA := math.Log(2.0/(1.0-quotaForPureTransaction/quotaLimit)-1.0) / maxHeightGap / minPledgeAmount
	fmt.Printf("paramA      = new(big.Float).SetPrec(precForFloat).SetFloat64(%v)\n", paramA)
	// Pledge no Vite Token, calc PoW for default difficulty, gets quota for a pure transfer transaction
	defaultDifficulty := float64(0xffffffc000000000)
	paramB := math.Log(2.0/(1.0-quotaForPureTransaction/quotaLimit)-1.0) / defaultDifficulty
	fmt.Printf("paramB      = new(big.Float).SetPrec(precForFloat).SetFloat64(%v)\n", paramB)

	fmt.Printf("sectionList = []*big.Float{ // Section list of x value in e**x\nnew(big.Float).SetPrec(precForFloat).SetFloat64(0.0),\n")
	q := 0.0
	index := 0
	for {
		index = index + 1
		q = q + quotaForPureTransaction
		if q >= quotaLimit {
			break
		}
		gapLow := math.Log(2.0/(1.0-q/quotaLimit) - 1.0)
		fmt.Printf("new(big.Float).SetPrec(precForFloat).SetFloat64(%v),\n", gapLow)
	}
	fmt.Printf("}\n")
}

func TestCalcQuotaForPoW(t *testing.T) {
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetInt(DefaultDifficulty)
	tmpFLoat.Mul(tmpFLoat, paramB)
	x.Add(x, tmpFLoat)
	quotaTotal := uint64(getIndexInSection(x)) * quotaForSection
	if quotaTotal != TxGas {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaTotal)
	}
}

func TestCalcQuotaForMinPledge(t *testing.T) {
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(86400)
	x.Mul(tmpFLoat, paramA)
	tmpFLoat.SetUint64(1e19)
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW < TxGas {
		t.Fatalf("gain quota pledge minimum Vite Token not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}

func TestCalcQuotaForMaxPledge(t *testing.T) {
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(86400)
	x.Mul(tmpFLoat, paramA)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	tmpFLoat.SetInt(viteTotalSupply)
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW != TxGas*uint64(len(sectionList)-1) {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}
