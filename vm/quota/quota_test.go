package quota

import (
	"fmt"
	"github.com/vitelabs/go-vite/vm/util"
	"math"
	"math/big"
	"testing"
)

// quota = Qm * (1 - 2 / (1 + e**(paramPledge * Hgap * pledgeAmount + paramPoW * difficulty)))
// Qm is decided by total quota used during past 3600 snapshot blocks,
// using 1 million for the sake of simplicity in test net.
// Default difficulty is 0xffffffc000000000.
func TestCalcLogisticQuotaParam(t *testing.T) {
	InitQuotaConfig(false)
	quotaLimit := 1000000.0
	defaultSection := nodeConfig.sectionList[1]

	// Pledge minimum amount of Vite Token, calc no PoW, wait for longest block height, gets quota for a pure transfer transaction
	// maxHeightGap := 86400.0
	// Pledge minimum amount of Vite Token, calc no PoW, wait for one snapshot block, gets quota for a pure transfer transaction
	maxHeightGap, _ := new(big.Float).SetPrec(precForFloat).SetString("1.0")
	minPledgeAmount, _ := new(big.Float).SetPrec(precForFloat).SetString("1.0e21")
	floatTmp := new(big.Float).SetPrec(precForFloat)
	floatTmp.Quo(defaultSection, maxHeightGap)
	floatTmp.Quo(floatTmp, minPledgeAmount)
	fmt.Printf("paramA      = new(big.Float).SetPrec(precForFloat).SetFloat64(%v)\n", floatTmp.String())
	// Pledge no Vite Token, calc PoW for default difficulty, gets quota for a pure transfer transaction
	//defaultDifficulty, _ := new(big.Float).SetPrec(precForFloat).SetString("0x000000000000FFFF")
	defaultDifficulty, _ := new(big.Float).SetPrec(precForFloat).SetString("0xffffffc000000000")
	floatTmp.Quo(defaultSection, defaultDifficulty)
	fmt.Printf("paramB      = new(big.Float).SetPrec(precForFloat).SetFloat64(%v)\n", floatTmp.String())

	fmt.Printf("sectionList = []*big.Float{ // Section list of x value in e**x\nnew(big.Float).SetPrec(precForFloat).SetFloat64(0.0),\n")
	q := 0.0
	index := 0
	for {
		index = index + 1
		q = q + 21000.0
		if q >= quotaLimit {
			break
		}
		gapLow := math.Log(2.0/(1.0-q/quotaLimit) - 1.0)
		fmt.Printf("new(big.Float).SetPrec(precForFloat).SetFloat64(%v),\n", gapLow)
	}
	fmt.Printf("}\n")
}

func TestCalcQuotaForPoWTest(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	difficulty := float64(0x000000000000FFFF)
	tmpFLoat.SetFloat64(difficulty)
	tmpFLoat.Mul(tmpFLoat, QuotaParamTest.paramB)
	x.Add(x, tmpFLoat)
	quotaTotal := uint64(getIndexInSection(x)) * quotaForSection
	if quotaTotal != util.TxGas {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaTotal)
	}
}

func TestCalcQuotaForMinPledgeTest(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(1)
	x.Mul(tmpFLoat, QuotaParamTest.paramA)
	tmpFLoat.SetInt(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)))
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW != util.TxGas {
		t.Fatalf("gain quota pledge minimum Vite Token not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}

func TestCalcQuotaForMaxPledgeTest(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(1)
	x.Mul(tmpFLoat, QuotaParamTest.paramA)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	tmpFLoat.SetInt(viteTotalSupply)
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW != util.TxGas*uint64(len(nodeConfig.sectionList)-1) {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}

func TestCalcQuotaForPoWMainNet(t *testing.T) {
	InitQuotaConfig(false)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	difficulty := float64(0xffffffc000000000)
	tmpFLoat.SetFloat64(difficulty)
	tmpFLoat.Mul(tmpFLoat, QuotaParamMainNet.paramB)
	x.Add(x, tmpFLoat)
	quotaTotal := uint64(getIndexInSection(x)) * quotaForSection
	if quotaTotal != util.TxGas {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaTotal)
	}
}

func TestCalcQuotaForMinPledgeMainNet(t *testing.T) {
	InitQuotaConfig(false)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(1)
	x.Mul(tmpFLoat, QuotaParamMainNet.paramA)
	tmpFLoat.SetInt(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)))
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW != util.TxGas {
		t.Fatalf("gain quota pledge minimum Vite Token not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}

func TestCalcQuotaForMaxPledgeMainNet(t *testing.T) {
	InitQuotaConfig(false)
	x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	tmpFLoat.SetUint64(1)
	x.Mul(tmpFLoat, QuotaParamMainNet.paramA)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	tmpFLoat.SetInt(viteTotalSupply)
	x.Mul(tmpFLoat, x)
	quotaWithoutPoW := uint64(getIndexInSection(x)) * quotaForSection
	if quotaWithoutPoW != util.TxGas*uint64(len(nodeConfig.sectionList)-1) {
		t.Fatalf("gain quota by calc PoW not enough to create a transaction, got %v", quotaWithoutPoW)
	}
}

func TestQuotaSection(t *testing.T) {
	x := new(big.Float).SetPrec(precForFloat)
	pledgeMin := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))
	for i := 0; i < len(nodeConfig.sectionList); i++ {
		x.SetInt(pledgeMin)
		x.Mul(x, QuotaParamTest.paramA)
		x.Quo(nodeConfig.sectionList[i], x)
		f, _ := x.Float64()
		fmt.Printf("pledgeAmount:1000 vite, wait time: %v, quotaForTx: %v\n", math.Ceil(f), i)
	}
}
