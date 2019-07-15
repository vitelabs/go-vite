package quota

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"math"
	"math/big"
	"strconv"
	"testing"
)

func TestPrintParamAndSectionList(t *testing.T) {
	quotaLimit := 1000000.0
	sectionList := make([]*big.Float, 0)
	fmt.Printf("sectionStrList = []string{\n")
	q := 0.0
	index := 0
	for {
		if q >= quotaLimit {
			break
		}
		gapLow := math.Log(2.0/(1.0-q/quotaLimit) - 1.0)

		fmt.Printf("\t\"%v\", \n", gapLow)
		sectionList = append(sectionList, new(big.Float).SetPrec(precForFloat).SetFloat64(gapLow))
		index = index + 1
		q = q + 280
	}
	fmt.Printf("}\n")

	defaultSectionForPledge := sectionList[75]
	defaultSectionForPoW := sectionList[75]

	floatTmp := new(big.Float).SetPrec(precForFloat)

	pledgeAmountForOneTpsMainnet, _ := new(big.Float).SetPrec(precForFloat).SetString("9999")
	pledgeAmountForOneTpsMainnet.Mul(pledgeAmountForOneTpsMainnet, new(big.Float).SetPrec(precForFloat).SetInt(util.AttovPerVite))
	floatTmp.Quo(defaultSectionForPledge, pledgeAmountForOneTpsMainnet)
	paramaForMainnet := floatTmp.String()

	defaultDifficultyForMainnet := new(big.Float).SetPrec(precForFloat).SetUint64(67108862)
	floatTmp.Quo(defaultSectionForPoW, defaultDifficultyForMainnet)
	parambForMainnet := floatTmp.String()

	fmt.Printf("QuotaParamMainnet  = NewQuotaParams(\"%v\", \"%v\")\n", paramaForMainnet, parambForMainnet)

	pledgeAmountForOneTpsTestnet, _ := new(big.Float).SetPrec(precForFloat).SetString("10")
	pledgeAmountForOneTpsTestnet.Mul(pledgeAmountForOneTpsTestnet, new(big.Float).SetPrec(precForFloat).SetInt(util.AttovPerVite))
	floatTmp.Quo(defaultSectionForPledge, pledgeAmountForOneTpsTestnet)
	paramaForTestnet := floatTmp.String()

	defaultDifficultyForTestnet := new(big.Float).SetPrec(precForFloat).SetUint64(65534)
	floatTmp.Quo(defaultSectionForPoW, defaultDifficultyForTestnet)
	parambForTestnet := floatTmp.String()

	fmt.Printf("QuotaParamTestnet  = NewQuotaParams(\"%v\", \"%v\")\n", paramaForTestnet, parambForTestnet)
}

func TestPrintPledgeAmountSection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false, false)
	p := nodeConfig.paramA
	fmt.Printf("pledgeAmountListMainnet = []string{\n")
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextPledgeAmount(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("\"%v\", \n", amount.String())
	}
	fmt.Printf("}\n")

	InitQuotaConfig(false, true)
	p = nodeConfig.paramA
	fmt.Printf("pledgeAmountListTestnet = []string{\n")
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextPledgeAmount(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("\"%v\", \n", amount.String())
	}
	fmt.Printf("}\n")
}

func TestPrintDifficultySection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false, false)
	p := nodeConfig.paramB
	resultmainnet := "difficultyListMainnet = []*big.Int{"

	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		resultmainnet = resultmainnet + "big.NewInt(" + amount.String() + "), "
	}
	resultmainnet = resultmainnet + "}"
	fmt.Println(resultmainnet)

	InitQuotaConfig(false, true)
	p = nodeConfig.paramB
	resulttestnet := "difficultyListTestnet = []*big.Int{"
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		resulttestnet = resulttestnet + "big.NewInt(" + amount.String() + "), "
	}
	resulttestnet = resulttestnet + "}"
	fmt.Println(resulttestnet)
}

func TestCheckNodeConfig(t *testing.T) {
	InitQuotaConfig(false, false)
	l := len(nodeConfig.sectionList)
	if len(nodeConfig.pledgeAmountList) != l || len(nodeConfig.difficultyList) != l {
		t.Fatalf("main net node config param error")
	}
	checkFloatList(nodeConfig.sectionList, "section list", t)
	checkList(nodeConfig.pledgeAmountList, "mainnet pledge amount", t)
	checkList(nodeConfig.difficultyList, "mainnet difficulty", t)
	InitQuotaConfig(false, true)
	l = len(nodeConfig.sectionList)
	if len(nodeConfig.pledgeAmountList) != l || len(nodeConfig.difficultyList) != l {
		t.Fatalf("main net node config param error")
	}
	checkList(nodeConfig.pledgeAmountList, "testnet pledge amount", t)
	checkList(nodeConfig.difficultyList, "testnet difficulty", t)
}

func checkList(list []*big.Int, s string, t *testing.T) {
	lastAmount := list[0]
	for _, amount := range list[1:] {
		if lastAmount.Cmp(amount) > 0 {
			t.Fatalf("invalid " + s + " list")
		}
		lastAmount = amount
	}
}

func checkFloatList(list []*big.Float, s string, t *testing.T) {
	lastAmount := list[0]
	for _, amount := range list[1:] {
		if lastAmount.Cmp(amount) > 0 {
			t.Fatalf("invalid " + s + " list")
		}
		lastAmount = amount
	}
}

func getNextPledgeAmount(bi *big.Int, p *big.Float, target *big.Float, tmp *big.Float) *big.Int {
	bi.Quo(bi, util.AttovPerVite)
	bi.Mul(bi, util.AttovPerVite)
	for {
		tmp = tmp.SetInt(bi)
		tmp = tmp.Mul(tmp, p)
		if tmp.Cmp(target) < 0 {
			bi = bi.Add(bi, util.AttovPerVite)
		} else {
			break
		}
	}
	return bi
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

func TestPrintUTPS(t *testing.T) {
	InitQuotaConfig(false, false)
	index := 75
	for {
		if index >= len(nodeConfig.pledgeAmountList) {
			break
		}
		fmt.Printf("| $(%v, %v]$ | %v | %v | %v | %v | %v |\n",
			nodeConfig.sectionList[index-75], nodeConfig.sectionList[index],
			index*21000,
			index/75,
			index,
			nodeConfig.pledgeAmountList[index],
			nodeConfig.difficultyList[index/75],
		)
		index += 75
	}
}

func TestPrintQuotaTable(t *testing.T) {
	InitQuotaConfig(false, true)
	index := 75
	for {
		if index >= len(nodeConfig.pledgeAmountList) {
			break
		}
		fmt.Printf("%v\t%v\t%v\t%v\n",
			index/75*21000,
			index/75,
			nodeConfig.pledgeAmountList[index],
			nodeConfig.difficultyList[index/75],
		)
		index += 75
	}
}

func TestPrintQuotaUnder1UTPS(t *testing.T) {
	InitQuotaConfig(false, false)
	for i := 1; i < 75; i = i + 1 {
		pledgeAmount := new(big.Int).Quo(nodeConfig.pledgeAmountList[i], util.AttovPerVite)
		fmt.Printf("| $(%v, %v]$ | %v | %v/75 | %v | %v | %v |\n", nodeConfig.sectionList[i-1], nodeConfig.sectionList[i], uint64(i)*quotaForSection, i, i, pledgeAmount, nodeConfig.difficultyList[i])
	}
}

func TestPrintQuota(t *testing.T) {
	InitQuotaConfig(false, false)
	for i := 75; i < len(nodeConfig.sectionList); i = i + 75 {
		pledgeAmount := new(big.Int).Quo(nodeConfig.pledgeAmountList[i], util.AttovPerVite)
		fmt.Printf("| $(%v, %v]$ | %v | %v | %v | %v | %v |\n", nodeConfig.sectionList[i-75], nodeConfig.sectionList[i], uint64(i)*quotaForSection, i/75, i, pledgeAmount, nodeConfig.difficultyList[i])
	}
}

//Qc(x) = 1, if x<=T1
//      = a-exp(bx-bT1), if T1 <x <= T2
//      = exp(cT2-cx)-d, if T2 <x
// Qc(T1) = 1
// Qc(T2) = 0.01
// Qc(T3) = 134/(425000000)
func TestPrintQcParams(t *testing.T) {
	/*t1 := uint64(50 * 74 * 21000)
	t2 := uint64(100 * 74 * 21000)
	t3 := uint64(500 * 74 * 21000)
	qcT1 := float64(1)
	qcT2 := float64(0.01)
	qcT3 := float64(0.000000536)*/

	t1 := uint64(50)
	t2 := uint64(100)
	t3 := uint64(500)
	qcT1 := float64(1)
	qcT2 := float64(0.01)
	qcT3 := float64(0.00001)
	a := 1 + qcT1
	b := math.Log(a-qcT2) / float64(t2-t1)
	d := 1 - qcT2
	c := math.Log(qcT3+d) / (float64(t2) - float64(t3))
	fmt.Println("a", a, "b", b, "c", c, "d", d)
	// a=2 b=6.112894154022806e-07 c =1.254284763124381e-08 d=0.9
	/*lastNum := uint64(1000000000000000000)
	for x := uint64(0); x <= t3; x = x + 21000*74 {
		newNum := uint64(testCalcQc(a, b, c, d, x) * 1e18)
		fmt.Println(x/21000, x, newNum, newNum-lastNum)
		lastNum = newNum
	}*/
	gap := uint64(21000 * 74)

	fmt.Println("qcIndexMinMainnet uint64 = " + strconv.Itoa(int(t2/gap)))
	fmt.Println("qcIndexMaxMainnet uint64 = " + strconv.Itoa(int(t3/gap)))
	for x := t1 + gap; x <= t3; x = x + gap {
		newNum := uint64(testCalcQc(a, b, c, d, t1, t2, x) * 1e18)
		fmt.Printf("%v: big.NewInt(%v),\n", x/gap, newNum)
	}
	fmt.Println("}")
}

func testCalcQc(a, b, c, d float64, t1, t2, x uint64) float64 {
	if x <= t1 {
		return float64(1)
	} else if x <= t2 {
		return a - math.Exp(b*float64(x)-b*float64(t1))
	} else {
		return math.Exp(c*float64(t2)-c*float64(x)) - d
	}
}

func TestPrintQcTable(t *testing.T) {
	var pledgeAmountMax = new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	InitQuotaConfig(false, false)
	pledgeAmount := new(big.Int).Set(nodeConfig.pledgeAmountList[1])
	for i := nodeConfig.qcIndexMin; i <= nodeConfig.qcIndexMax; i = i + 1 {
		qc, ok := nodeConfig.qcMap[i]
		if !ok {
			t.Fatalf("index %v not found", i)
		}
		tmp := new(big.Float).SetPrec(18).SetInt(qc)
		tmp.Quo(tmp, new(big.Float).SetPrec(18).SetInt(qcDivision))
		p, _ := calcPledgeTargetParam(qc, true, pledgeAmount)
		pledgeAmountStr := "-"
		if p.Cmp(pledgeAmountMax) < 0 {
			pledgeAmountStr = getViteByAttov(p).String()
		}
		fmt.Printf("| %v | %v | %v |\n", i, tmp.Text('g', 18), pledgeAmountStr)
	}
}

func TestPrintPledgeTableByQc(t *testing.T) {
	InitQuotaConfig(false, false)
	qc51 := new(big.Int).Set(nodeConfig.qcMap[51])
	qc100 := new(big.Int).Set(nodeConfig.qcMap[100])
	qc200 := new(big.Int).Set(nodeConfig.qcMap[200])
	for i := 1; i < len(nodeConfig.pledgeAmountList); {
		pledgeAmount := new(big.Int).Set(nodeConfig.pledgeAmountList[i])
		difficulty := new(big.Int).Set(nodeConfig.difficultyList[i])
		utpsStr := ""
		if i < 75 {
			utpsStr = strconv.Itoa(i) + "/75"
		} else {
			utpsStr = strconv.Itoa(i / 75)
		}
		fmt.Printf("| %v | %v | %v | %v | %v | %v | %v | %v | %v | %v |\n",
			uint64(i)*quotaForSection, utpsStr,
			new(big.Int).Div(pledgeAmount, big.NewInt(1e18)), difficulty,
			getPledgeAmountByAttov(qc51, pledgeAmount), getPledgeDifficultyByAttov(qc51, difficulty),
			getPledgeAmountByAttov(qc100, pledgeAmount), getPledgeDifficultyByAttov(qc100, difficulty),
			getPledgeAmountByAttov(qc200, pledgeAmount), getPledgeDifficultyByAttov(qc200, difficulty),
		)
		if i < 75 {
			i = i + 1
		} else {
			i = i + 75
		}
	}
}

func getPledgeAmountByAttov(qc *big.Int, param *big.Int) *big.Int {
	target, err := calcPledgeTargetParam(qc, true, param)
	if err == nil {
		return getViteByAttov(target)
	}
	return nil
}
func getPledgeDifficultyByAttov(qc *big.Int, param *big.Int) *big.Int {
	target, _ := calcPledgeTargetParam(qc, true, param)
	return target
}
func getViteByAttov(i *big.Int) *big.Int {
	return new(big.Int).Div(i, big.NewInt(1e18))
}
func getUtpsStr(q uint64) string {
	return strconv.FormatFloat(float64(q)/float64(21000), 'g', 4, 64)
}
