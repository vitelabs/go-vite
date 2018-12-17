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
	minPledgeAmount, _ := new(big.Float).SetPrec(precForFloat).SetString("1.0e22")
	floatTmp := new(big.Float).SetPrec(precForFloat)
	floatTmp.Quo(defaultSection, maxHeightGap)
	floatTmp.Quo(floatTmp, minPledgeAmount)
	fmt.Printf("paramA      = new(big.Float).SetPrec(precForFloat).SetFloat64(%v)\n", floatTmp.String())
	// Pledge no Vite Token, calc PoW for default difficulty, gets quota for a pure transfer transaction
	//defaultDifficulty, _ := new(big.Float).SetPrec(precForFloat).SetString("0x000000000000FFFF")
	defaultDifficulty := new(big.Float).SetPrec(precForFloat).SetUint64(67108863)
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
	difficulty := float64(67108863)
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
	tmpFLoat.SetInt(new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e18)))
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

func TestPledgeQuotaSection(t *testing.T) {
	InitQuotaConfig(false)
	x := new(big.Float).SetPrec(precForFloat)
	pledgeMin := new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e18))
	for i := 0; i < len(nodeConfig.sectionList); i++ {
		x.SetInt(pledgeMin)
		x.Mul(x, nodeConfig.paramA)
		x.Quo(nodeConfig.sectionList[i], x)
		f, _ := x.Float64()
		fmt.Printf("pledgeAmount:10000 vite, wait time: %v, quotaForTx: %v\n", math.Ceil(f), i)
	}
}

func TestPoWQuotaSection(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(precForFloat)
	for i := 0; i < len(nodeConfig.sectionList); i++ {
		x.Set(nodeConfig.paramB)
		x.Quo(nodeConfig.sectionList[i], x)
		difficulty, _ := new(big.Int).SetString(x.Text('f', 0), 0)
		//target := pow.DifficultyToTarget(difficulty)
		//fmt.Printf("pow difficulty: %v, quota: %v\n", difficulty, i*21000)
		fmt.Printf("%v\n", difficulty)
	}
}

func TestPoWQuota(t *testing.T) {
	InitQuotaConfig(false)
	list := []string{"0", "67108864", "134276096", "201564160", "269029376", "336736256", "404742144", "473120768", "541929472", "611241984", "681119744", "751652864", "822910976", "894976000", "967946240", "1041903616", "1116962816", "1193222144", "1270800384", "1349836800", "1430462464", "1512824832", "1597120512", "1683513344", "1772216320", "1863491584", "1957568512", "2054791168", "2155479040", "2260041728", "2368962560", "2482749440", "2602090496", "2727755776", "2860646400", "3001925632", "3153051648", "3315826688", "3492593664", "3686514688", "3901882368", "4144775168", "4424400896", "4755193856", "5162500096", "5696520192", "6482067456", "8034975744"}
	x := new(big.Float).SetPrec(precForFloat)
	for i, str := range list {
		difficulty, _ := new(big.Int).SetString(str, 10)
		x.SetInt(difficulty)
		x.Mul(x, nodeConfig.paramB)
		if getIndexInSection(x) != i {
			fmt.Printf("get quota by pow failed, difficulty = %v", str)
		}
	}
}

func TestCalcPoWDifficultyMainNet(t *testing.T) {
	InitQuotaConfig(false)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	for i, difficulty := range difficultyListMainNet {
		tmpFLoat.SetInt(difficulty)
		tmpFLoat.Mul(tmpFLoat, nodeConfig.paramB)
		q := calcQuotaInSection(tmpFLoat)
		if q != uint64(i)*quotaForSection {
			t.Fatalf("calc pow difficulty of main net failed, %v: %v", i, difficulty)
		}
	}
}
func TestCalcPoWDifficultyTest(t *testing.T) {
	InitQuotaConfig(true)
	tmpFLoat := new(big.Float).SetPrec(precForFloat)
	for i, difficulty := range difficultyListTest {
		tmpFLoat.SetInt(difficulty)
		tmpFLoat.Mul(tmpFLoat, nodeConfig.paramB)
		q := calcQuotaInSection(tmpFLoat)
		if q != uint64(i)*quotaForSection {
			t.Fatalf("calc pow difficulty of main net failed, %v: %v", i, difficulty)
		}
	}
}
