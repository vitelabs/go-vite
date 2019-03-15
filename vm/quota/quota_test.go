package quota

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"math"
	"math/big"
	"testing"
)

func TestCalcParamAndSectionList(t *testing.T) {
	quotaLimit := 75 * 1000000.0
	sectionList := make([]*big.Float, 0)
	fmt.Printf("sectionStrList = []string{\n")
	q := 0.0
	index := 0
	wapperIndex := 0
	for {
		if q >= quotaLimit {
			break
		}
		gapLow := math.Log(2.0/(1.0-q/quotaLimit) - 1.0)

		fmt.Printf("\t\"%v\", ", gapLow)
		wapperIndex = wapperIndex + 1
		if wapperIndex == 75 {
			fmt.Printf("\n")
			wapperIndex = 0
		}
		sectionList = append(sectionList, new(big.Float).SetPrec(precForFloat).SetFloat64(gapLow))
		index = index + 1
		q = q + 21000.0
	}
	fmt.Printf("}\n")

	defaultSection := sectionList[1]

	floatTmp := new(big.Float).SetPrec(precForFloat)

	pledgeAmountForOneTpsMainnet, _ := new(big.Float).SetPrec(precForFloat).SetString("9999")
	floatTmp.Quo(defaultSection, pledgeAmountForOneTpsMainnet)
	paramaForMainnet := floatTmp.String()

	defaultDifficultyForMainnet := new(big.Float).SetPrec(precForFloat).SetUint64(67108862)
	floatTmp.Quo(defaultSection, defaultDifficultyForMainnet)
	parambForMainnet := floatTmp.String()

	fmt.Printf("QuotaParamMainnet  = NewQuotaParams(\"%v\", \"%v\")\n", paramaForMainnet, parambForMainnet)

	pledgeAmountForOneTpsTestnet, _ := new(big.Float).SetPrec(precForFloat).SetString("10")
	floatTmp.Quo(defaultSection, pledgeAmountForOneTpsTestnet)
	paramaForTestnet := floatTmp.String()

	defaultDifficultyForTestnet := new(big.Float).SetPrec(precForFloat).SetUint64(65534)
	floatTmp.Quo(defaultSection, defaultDifficultyForTestnet)
	parambForTestnet := floatTmp.String()

	fmt.Printf("QuotaParamTestnet  = NewQuotaParams(\"%v\", \"%v\")\n", paramaForTestnet, parambForTestnet)
}

func TestCalcPledgeAmountSection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false)
	p := nodeConfig.paramA
	fmt.Printf("pledgeAmountListMainnet = []*big.Int{\n")
	wapperIndex := 0
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wapperIndex == 75 {
			fmt.Printf("\n")
			wapperIndex = 0
		} else {
			wapperIndex = wapperIndex + 1
		}
	}
	fmt.Printf("}\n")

	InitQuotaConfig(true)
	p = nodeConfig.paramA
	fmt.Printf("pledgeAmountListTestnet = []*big.Int{\n")
	wapperIndex = 0
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wapperIndex == 75 {
			fmt.Printf("\n")
			wapperIndex = 0
		} else {
			wapperIndex = wapperIndex + 1
		}
	}
	fmt.Printf("}\n")
}

func TestCalcDifficultySection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false)
	p := nodeConfig.paramB
	fmt.Printf("difficultyListMainnet = []*big.Int{\n")
	wapperIndex := 0
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wapperIndex == 75 {
			fmt.Printf("\n")
			wapperIndex = 0
		} else {
			wapperIndex = wapperIndex + 1
		}
	}
	fmt.Printf("}\n")

	InitQuotaConfig(true)
	p = nodeConfig.paramB
	fmt.Printf("difficultyListTestnet = []*big.Int{\n")
	wapperIndex = 0
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wapperIndex == 75 {
			fmt.Printf("\n")
			wapperIndex = 0
		} else {
			wapperIndex = wapperIndex + 1
		}
	}
	fmt.Printf("}\n")
}

func TestCheckNodeConfig(t *testing.T) {
	InitQuotaConfig(false)
	l := len(nodeConfig.sectionList)
	if len(nodeConfig.pledgeAmountList) != l || len(nodeConfig.difficultyList) != l {
		t.Fatalf("main net node config param error")
	}
	InitQuotaConfig(true)
	l = len(nodeConfig.sectionList)
	if len(nodeConfig.pledgeAmountList) != l || len(nodeConfig.difficultyList) != l {
		t.Fatalf("main net node config param error")
	}
}

func getNextBigInt(bi *big.Int, p *big.Float, target *big.Float, tmp *big.Float) *big.Int {
	for {
		tmp = tmp.SetInt(bi)
		tmp = tmp.Mul(tmp, p)
		if tmp.Cmp(target) < 0 {
			bi = bi.Add(bi, helper.Big1)
		} else {
			break
		}
	}
	return bi
}

func TestCalcQuotaV2(t *testing.T) {
	// TODO
}
