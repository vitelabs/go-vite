package quota

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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

	defaultSection := sectionList[75]

	floatTmp := new(big.Float).SetPrec(precForFloat)

	pledgeAmountForOneTpsMainnet, _ := new(big.Float).SetPrec(precForFloat).SetString("9999")
	floatTmp.Quo(defaultSection, pledgeAmountForOneTpsMainnet)
	paramaForMainnet := floatTmp.String()

	defaultDifficultyForMainnet := new(big.Float).SetPrec(precForFloat).SetUint64(67108862)
	floatTmp.Quo(defaultSection, defaultDifficultyForMainnet)
	parambForMainnet := floatTmp.String()

	fmt.Printf("QuotaParamMainnet  = NewQuotaParams(\"%v\", \"%v\")\n", paramaForMainnet, parambForMainnet)

	pledgeAmountForOneTpsTestnet, _ := new(big.Float).SetPrec(precForFloat).SetString("9")
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

	InitQuotaConfig(false, false)
	p := nodeConfig.paramA
	fmt.Printf("pledgeAmountListMainnet = []*big.Int{\n")
	for wrapperIndex, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wrapperIndex%75 == 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("}\n")

	InitQuotaConfig(false, true)
	p = nodeConfig.paramA
	fmt.Printf("pledgeAmountListTestnet = []*big.Int{\n")
	for wrapperIndex, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wrapperIndex%75 == 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("}\n")
}

func TestCalcDifficultySection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false, false)
	p := nodeConfig.paramB
	fmt.Printf("difficultyListMainnet = []*big.Int{\n")
	for wrapperIndex, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wrapperIndex%75 == 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("}\n")

	InitQuotaConfig(false, true)
	p = nodeConfig.paramB
	fmt.Printf("difficultyListTestnet = []*big.Int{\n")
	for wrapperIndex, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), ", amount.String())
		if wrapperIndex%75 == 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("}\n")
}

func TestCheckNodeConfig(t *testing.T) {
	InitQuotaConfig(false, false)
	l := len(nodeConfig.sectionList)
	if len(nodeConfig.pledgeAmountList) != l || len(nodeConfig.difficultyList) != l {
		t.Fatalf("main net node config param error")
	}
	InitQuotaConfig(false, true)
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

type testQuotaDb struct {
	addr                  types.Address
	quotaUsed, blockCount uint64
	unconfirmedBlockList  []*ledger.AccountBlock
}

func (db *testQuotaDb) Address() *types.Address {
	return &db.addr
}
func (db *testQuotaDb) GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64) {
	return db.quotaUsed, db.blockCount
}
func (db *testQuotaDb) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return db.unconfirmedBlockList
}

func TestCalcPoWDifficulty(t *testing.T) {
	testCases := []struct {
		quotaRequired uint64
		q             types.Quota
		pledgeAmount  *big.Int
		difficulty    *big.Int
		err           error
		name          string
	}{
		{1000001, types.NewQuota(100000000, 0, 0), big.NewInt(0), nil, errors.New("quota limit for block reached"), "block_quota_limit_reached"},
		{21000, types.NewQuota(74970001, 74970001, 0), big.NewInt(0), nil, errors.New("quota limit for account reached"), "account_quota_limit_reached"},
		{21000, types.NewQuota(74970002, 74970001, 0), big.NewInt(0), nil, errors.New("quota limit for account reached"), "account_quota_limit_reached2"},
		{21000, types.NewQuota(0, 0, 0), big.NewInt(0), big.NewInt(894654), nil, "no_pledge_quota"},
		{21000, types.NewQuota(21000, 0, 0), big.NewInt(134), big.NewInt(0), nil, "pledge_quota_enough"},
		{22000, types.NewQuota(21000, 0, 0), big.NewInt(134), big.NewInt(889959), nil, "use_both"},
		{21000, types.NewQuota(0, 0, 0), big.NewInt(134), big.NewInt(0), nil, "total_quota_not_exact"},
		{1000000, types.NewQuota(21000, 21000, 21000), big.NewInt(134), big.NewInt(42941413), nil, "total_quota_not_exact"},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		difficulty, err := CalcPoWDifficulty(testCase.quotaRequired, testCase.q, testCase.pledgeAmount)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v CalcPoWDifficulty failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && difficulty.Cmp(testCase.difficulty) != 0 {
			t.Fatalf("%v CalcPoWDifficulty failed, difficulty not match, expected %v, got %v", testCase.name, testCase.difficulty, difficulty)
		}
	}
}

func TestCanPoW(t *testing.T) {
	testCases := []struct {
		blockList []*ledger.AccountBlock
		result    bool
		name      string
	}{
		{[]*ledger.AccountBlock{}, true, "no_blocks"},
		{[]*ledger.AccountBlock{{Nonce: []byte{1}}}, false, "cannot_calc_pow1"},
		{[]*ledger.AccountBlock{{}, {Nonce: []byte{1}}}, false, "cannot_calc_pow2"},
		{[]*ledger.AccountBlock{{}}, true, "can_calc_pow1"},
		{[]*ledger.AccountBlock{{}, {}}, true, "can_calc_pow2"},
	}
	for _, testCase := range testCases {
		db := &testQuotaDb{types.Address{}, 0, 0, testCase.blockList}
		result, _ := CanPoW(db)
		if result != testCase.result {
			t.Fatalf("%v CanPoW failed, result not match, expected %v, got %v", testCase.name, testCase.result, result)
		}
	}
}

func TestCalcQuotaV3(t *testing.T) {
	testCases := []struct {
		addr                                           types.Address
		pledgeAmount                                   *big.Int
		difficulty                                     *big.Int
		usedQuota, blockCount                          uint64
		unconfirmedBlockList                           []*ledger.AccountBlock
		quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64
		err                                            error
		name                                           string
	}{
		{types.Address{}, big.NewInt(0), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			0, 0, 0, 0, nil, "no_quota",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(1),
			0, 0, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, 0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(894654),
			21000, 2, []*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			42000, 21000, 21000, 10500, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			21000, 0, 0, 0, nil, "get_quota_by_pledge1",
		},
		{types.Address{}, big.NewInt(267), big.NewInt(0),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			42000, 0, 21000, 21000, nil, "get_quota_by_pledge2",
		},
		{types.Address{}, big.NewInt(400), big.NewInt(0),
			42001, 2, []*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			63000, 0, 42001, 21000, nil, "get_quota_by_pledge3",
		},
		{types.Address{}, big.NewInt(135), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			21000, 0, 0, 0, nil, "get_quota_by_pledge4",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(894654),
			0, 0, []*ledger.AccountBlock{},
			21000, 21000, 0, 0, nil, "get_quota_by_difficulty1",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(894654),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			42000, 21000, 21000, 21000, nil, "get_quota_by_difficulty2",
		},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, testCase.usedQuota, testCase.blockCount, testCase.unconfirmedBlockList}
		quotaTotal, quotaAddition, quotaUsed, quotaAvg, err := calcQuotaV3(db, testCase.addr, testCase.pledgeAmount, testCase.difficulty)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v calcQuotaV3 failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (quotaTotal != testCase.quotaTotal || quotaAddition != testCase.quotaAddition || quotaUsed != testCase.quotaUsed || quotaAvg != testCase.quotaAvg) {
			t.Fatalf("%v calcQuotaV3 failed, quota not match, expected (%v,%v,%v,%v), got (%v,%v,%v,%v)", testCase.name, testCase.quotaTotal, testCase.quotaAddition, testCase.quotaUsed, testCase.quotaAvg, quotaTotal, quotaAddition, quotaUsed, quotaAvg)
		}
	}
}
func BenchmarkCalcQuotaV3(b *testing.B) {
	InitQuotaConfig(false, false)
	addr := types.Address{}
	db := &testQuotaDb{addr, 21000, 1, []*ledger.AccountBlock{{Quota: 21000}}}
	pledgeAmount := big.NewInt(10000)
	difficulty := big.NewInt(894654)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calcQuotaV3(db, addr, pledgeAmount, difficulty)
	}
}

func TestCalcQuotaForBlock(t *testing.T) {
	testCases := []struct {
		addr                      types.Address
		pledgeAmount              *big.Int
		difficulty                *big.Int
		usedQuota, blockCount     uint64
		unconfirmedBlockList      []*ledger.AccountBlock
		quotaTotal, quotaAddition uint64
		err                       error
		name                      string
	}{
		{types.Address{}, big.NewInt(0), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			0, 0, nil, "no_quota",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(1),
			0, 0, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(894654),
			21000, 2, []*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			21000, 21000, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge1",
		},
		{types.Address{}, big.NewInt(267), big.NewInt(0),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			21000, 0, nil, "get_quota_by_pledge2",
		},
		{types.Address{}, big.NewInt(400), big.NewInt(0),
			42001, 2, []*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			20999, 0, nil, "get_quota_by_pledge3",
		},
		{types.Address{}, big.NewInt(135), big.NewInt(0),
			0, 0, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge4",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(894654),
			0, 0, []*ledger.AccountBlock{},
			21000, 21000, nil, "get_quota_by_difficulty1",
		},
		{types.Address{}, big.NewInt(134), big.NewInt(894654),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			21000, 21000, nil, "get_quota_by_difficulty2",
		},
		{types.Address{}, big.NewInt(10), big.NewInt(0),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			0, 0, nil, "quota_total_less_than_used",
		},
		{types.Address{}, big.NewInt(10133), big.NewInt(0),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			1000000, 0, nil, "block_quota_limit_reached1",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(894654),
			21000, 1, []*ledger.AccountBlock{{Quota: 21000}},
			1000000, 21000, nil, "block_quota_limit_reached2",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(894654),
			0, 0, []*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached3",
		},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, testCase.usedQuota, testCase.blockCount, testCase.unconfirmedBlockList}
		quotaTotal, quotaAddition, err := CalcQuotaForBlock(db, testCase.addr, testCase.pledgeAmount, testCase.difficulty)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v TestCalcQuotaForBlock failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (quotaTotal != testCase.quotaTotal || quotaAddition != testCase.quotaAddition) {
			t.Fatalf("%v TestCalcQuotaForBlock failed, quota not match, expected (%v,%v), got (%v,%v)", testCase.name, testCase.quotaTotal, testCase.quotaAddition, quotaTotal, quotaAddition)
		}
	}
}

func TestCalcUTPS(t *testing.T) {
	InitQuotaConfig(false, false)
	index := 75
	for {
		if index >= len(nodeConfig.pledgeAmountList) {
			break
		}
		fmt.Printf("| $(%v, %v]$ | %v | %v | %v | %v |\n",
			nodeConfig.sectionList[index-75], nodeConfig.sectionList[index],
			index*21000,
			index/75,
			nodeConfig.pledgeAmountList[index],
			nodeConfig.difficultyList[index],
		)
		index += 75
	}
}

func TestCalcQuotaTable(t *testing.T) {
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
			nodeConfig.difficultyList[index],
		)
		index += 75
	}
}
