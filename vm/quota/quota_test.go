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

func TestPledgeWaitHeightSection(t *testing.T) {
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

func TestPledgeAmountSection(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(precForFloat)
	unit := new(big.Float).SetInt64(1e18)
	for i := 0; i < len(nodeConfig.sectionList); i++ {
		x.Set(nodeConfig.paramA)
		x.Quo(nodeConfig.sectionList[i], x)
		x.Quo(x, unit)
		f, _ := x.Float64()
		fmt.Printf("\"%v\",", int(math.Ceil(f)))
	}
}

func TestPledgeQuota(t *testing.T) {
	/*InitQuotaConfig(false)
	list := []string{"0", "10000", "20009", "30036", "40089", "50178", "60312", "70501", "80754", "91082", "101495", "112005", "122623", "133362", "144235", "155256", "166440", "177804", "189364", "201141", "213156", "225428", "237989", "250862", "264082", "277682", "291702", "306188", "321192", "336772", "353004", "369958", "387743", "406468", "426270", "447322", "469840", "494096", "520436", "549332", "581427", "617620", "659288", "708576", "769276", "848844", "965904", "1197301"}
	*/
	InitQuotaConfig(true)
	list := []string{"0", "11", "21", "31", "41", "51", "61", "71", "81", "92", "102", "113", "123", "134", "145", "156", "167", "178", "190", "202", "214", "226", "238", "251", "265", "278", "292", "307", "322", "337", "354", "370", "388", "407", "427", "448", "470", "495", "521", "550", "582", "618", "660", "709", "770", "849", "966", "1198"}
	x := new(big.Float).SetPrec(precForFloat)
	unit := new(big.Float).SetInt64(1e18)
	for i, str := range list {
		pledgeAmount, _ := new(big.Int).SetString(str, 10)
		x.SetInt(pledgeAmount)
		x.Mul(x, unit)
		x.Mul(x, nodeConfig.paramA)
		if got := getIndexInSection(x); got != i {
			fmt.Printf("get quota by pledge failed, pledgeAmount = %v, expected %v, got %v， section: %v\n", str, i, got, sectionStrList[i])
		}
	}
}

func TestPoWQuotaSection(t *testing.T) {
	InitQuotaConfig(true)
	x := new(big.Float).SetPrec(100)
	for i := 0; i < len(nodeConfig.sectionList); i++ {
		x.Set(nodeConfig.paramB)
		x.Quo(nodeConfig.sectionList[i], x)
		f, _ := x.Float64()
		fmt.Println(int(math.Ceil(f)))
		//difficulty, _ := new(big.Int).SetString(x.Text('f', 0), 10)
		//target := pow.DifficultyToTarget(difficulty)
		//fmt.Printf("pow difficulty: %v, quota: %v\n", difficulty, i*21000)
		//fmt.Printf("%v\n", difficulty)
	}
}

func TestPoWQuota(t *testing.T) {
	InitQuotaConfig(false)
	list := []string{"0", "67108864", "134276096", "201564160", "269029376", "336736256", "404742144", "473120768", "541929472", "611241984", "681119744", "751652864", "822910976", "894976000", "967946240", "1041903616", "1116962816", "1193222144", "1270800384", "1349836800", "1430462464", "1512824832", "1597120512", "1683513344", "1772216320", "1863491584", "1957568512", "2054791168", "2155479040", "2260041728", "2368962560", "2482757632", "2602090496", "2727755776", "2860646400", "3001933824", "3153051648", "3315826688", "3492593664", "3686514688", "3901882368", "4144775168", "4424400896", "4755193856", "5162500096", "5696520192", "6482067456", "8034975744"}
	/*
		InitQuotaConfig(true)
		list := []string{"0", "65535", "131127", "196836", "262720", "328838", "395250", "462024", "529220", "596904", "665147", "734024", "803612", "873984", "945240", "1017468", "1090768", "1165237", "1241000", "1318176", "1396912", "1477344", "1559656", "1644024", "1730656", "1819784", "1911656", "2006592", "2104928", "2207040", "2313408", "2424528", "2541072", "2663776", "2793552", "2931520", "3079104", "3238064", "3410672", "3600048", "3810368", "4047568", "4320608", "4643648", "5041440", "5562912", "6330048", "7846496"}
	*/
	x := new(big.Float).SetPrec(precForFloat)
	for i, str := range list {
		difficulty, _ := new(big.Int).SetString(str, 10)
		x.SetInt(difficulty)
		x.Mul(x, nodeConfig.paramB)
		if got := getIndexInSection(x); got != i {
			fmt.Printf("get quota by pow failed, difficulty = %v, expected %v, got %v， section: %v\n", str, i, got, sectionStrList[i])
		}
	}
}
