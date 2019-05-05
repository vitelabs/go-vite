package quota

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math"
	"math/big"
	"testing"
)

func TestCalcParamAndSectionList(t *testing.T) {
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

func TestCalcPledgeAmountSection(t *testing.T) {
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

func TestCalcDifficultySection(t *testing.T) {
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	tmpFloatForCalc := new(big.Float).SetPrec(precForFloat)

	InitQuotaConfig(false, false)
	p := nodeConfig.paramB
	fmt.Printf("difficultyListMainnet = []*big.Int{\n")
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), \n", amount.String())
	}
	fmt.Printf("}\n")

	InitQuotaConfig(false, true)
	p = nodeConfig.paramB
	fmt.Printf("difficultyListTestnet = []*big.Int{\n")
	for _, sec := range nodeConfig.sectionList {
		tmpFloat = tmpFloat.Quo(sec, p)
		amount, _ := tmpFloat.Int(nil)
		amount = getNextBigInt(amount, p, sec, tmpFloatForCalc)
		fmt.Printf("big.NewInt(%v), \n", amount.String())
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

type testQuotaDb struct {
	addr                 types.Address
	quotaList            []types.QuotaInfo
	unconfirmedBlockList []*ledger.AccountBlock
}

func (db *testQuotaDb) Address() *types.Address {
	return &db.addr
}
func (db *testQuotaDb) GetQuotaUsedList(address types.Address) []types.QuotaInfo {
	return db.quotaList
}
func (db *testQuotaDb) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	return db.unconfirmedBlockList
}

func TestCalcPoWDifficulty(t *testing.T) {
	testCases := []struct {
		quotaRequired uint64
		q             types.Quota
		difficulty    *big.Int
		err           error
		name          string
	}{
		{1000001, types.NewQuota(0, 0, 0, 0), nil, errors.New("quota limit for block reached"), "block_quota_limit_reached"},
		{21000, types.NewQuota(0, 0, 0, 0), big.NewInt(67108863), nil, "no_pledge_quota"},
		{22000, types.NewQuota(0, 0, 0, 0), big.NewInt(70689140), nil, "pledge_quota_not_enough"},
		{21000, types.NewQuota(0, 21000, 0, 0), big.NewInt(0), nil, "current_quota_enough"},
		{21000, types.NewQuota(0, 21001, 0, 0), big.NewInt(0), nil, "current_quota_enough"},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		difficulty, err := CalcPoWDifficulty(testCase.quotaRequired, testCase.q)
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
	addr := types.Address{}
	for _, testCase := range testCases {
		db := &testQuotaDb{addr, nil, testCase.blockList}
		result, _ := CanPoW(db, addr)
		if result != testCase.result {
			t.Fatalf("%v CanPoW failed, result not match, expected %v, got %v", testCase.name, testCase.result, result)
		}
	}
}

func TestCalcQuotaV3(t *testing.T) {
	testCases := []struct {
		addr                                                               types.Address
		pledgeAmount                                                       *big.Int
		difficulty                                                         *big.Int
		quotaInfoList                                                      []types.QuotaInfo
		unconfirmedList                                                    []*ledger.AccountBlock
		quotaTotal, pledgeQuota, quotaAddition, quotaUnconfirmed, quotaAvg uint64
		err                                                                error
		name                                                               string
	}{
		{types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			0, 0, 0, 0, 0, nil, "no_quota",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 21000, 0, 0, 0, nil, "new_pledge",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, 21000, 0, 0, nil, "new_pow",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{},
			210000, 21000, 0, 0, 0, nil, "pledge_1",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{
				{Quota: 10500, QuotaUsed: 10500},
			},
			199500, 21000, 0, 10500, 10500, nil, "pledge_2",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{
				{Quota: 10500, QuotaUsed: 10500},
				{Quota: 63000, QuotaUsed: 63000},
			},
			136500, 21000, 0, 73500, 36750, nil, "pledge_3",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
			},
			[]*ledger.AccountBlock{},
			136500, 21000, 0, 0, 36750, nil, "pledge_4",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
			},
			[]*ledger.AccountBlock{
				{Quota: 105000, QuotaUsed: 105000},
			},
			31500, 21000, 0, 105000, 59500, nil, "pledge_5",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
			},
			[]*ledger.AccountBlock{},
			31500, 21000, 0, 0, 59500, nil, "pledge_6",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
			},
			[]*ledger.AccountBlock{
				{Quota: 50000, QuotaUsed: 50000},
			},
			0, 0, 0, 0, 0, util.ErrInvalidUnconfirmedQuota, "pledge_7",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
			},
			[]*ledger.AccountBlock{
				{Quota: 31500, QuotaUsed: 31500},
			},
			0, 21000, 0, 31500, 52500, nil, "pledge_8",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 31500},
			},
			[]*ledger.AccountBlock{},
			21000, 21000, 0, 0, 52500, nil, "pledge_9",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 31500},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{},
			42000, 21000, 0, 0, 52500, nil, "pledge_10",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 31500},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{},
			63000, 21000, 21000, 0, 52500, nil, "pledge_and_pow",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 31500},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{
				{Quota: 11500, QuotaUsed: 31500, Nonce: []byte{1}},
			},
			0, 0, 0, 0, 0, util.ErrCalcPoWTwice, "can_not_pow",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 31500},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{
				{Quota: 31500, QuotaUsed: 31500},
			},
			31500, 21000, 21000, 31500, 48300, nil, "pledge_and_pow_2",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
				{BlockCount: 2, QuotaUsedTotal: 73500, QuotaTotal: 73500},
				{BlockCount: 1, QuotaUsedTotal: 105000, QuotaTotal: 105000},
				{BlockCount: 1, QuotaUsedTotal: 31500, QuotaTotal: 11500},
				{BlockCount: 0, QuotaUsedTotal: 0, QuotaTotal: 0},
			},
			[]*ledger.AccountBlock{
				{Quota: 31500, QuotaUsed: 31500},
			},
			41000, 21000, 21000, 31500, 48300, nil, "calc_quota_used",
		},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, updateUnconfirmedQuotaInfo(testCase.quotaInfoList, testCase.unconfirmedList), testCase.unconfirmedList}
		quotaTotal, pledgeQuota, quotaAddition, quotaUnconfirmed, quotaAvg, err := calcQuotaV3(db, testCase.addr, getPledgeAmount(testCase.pledgeAmount), testCase.difficulty)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v calcQuotaV3 failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (quotaTotal != testCase.quotaTotal || pledgeQuota != testCase.pledgeQuota || quotaAddition != testCase.quotaAddition || quotaUnconfirmed != testCase.quotaUnconfirmed || quotaAvg != testCase.quotaAvg) {
			t.Fatalf("%v calcQuotaV3 failed, quota not match, expected (%v,%v,%v,%v,%v), got (%v,%v,%v,%v,%v)", testCase.name, testCase.quotaTotal, testCase.pledgeQuota, testCase.quotaAddition, testCase.quotaUnconfirmed, testCase.quotaAvg, quotaTotal, pledgeQuota, quotaAddition, quotaUnconfirmed, quotaAvg)
		}
	}
}

func getPledgeAmount(amount *big.Int) *big.Int {
	return new(big.Int).Mul(amount, util.AttovPerVite)
}

func updateUnconfirmedQuotaInfo(quotaInfoList []types.QuotaInfo, unconfirmedList []*ledger.AccountBlock) []types.QuotaInfo {
	quotaInfo := types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0}
	for _, block := range unconfirmedList {
		quotaInfo.BlockCount = quotaInfo.BlockCount + 1
		quotaInfo.QuotaTotal = quotaInfo.QuotaTotal + block.Quota
		quotaInfo.QuotaUsedTotal = quotaInfo.QuotaUsedTotal + block.QuotaUsed
	}
	quotaInfoList = append(quotaInfoList, quotaInfo)
	return quotaInfoList
}

func BenchmarkCalcQuotaV3(b *testing.B) {
	InitQuotaConfig(false, false)
	addr := types.Address{}
	quotaInfoList := make([]types.QuotaInfo, 74)
	unConfirmedList := []*ledger.AccountBlock{
		{Quota: 10500, QuotaUsed: 10500},
	}
	db := &testQuotaDb{addr, updateUnconfirmedQuotaInfo(quotaInfoList, unConfirmedList), unConfirmedList}
	pledgeAmount := big.NewInt(10000)
	difficulty := big.NewInt(67108863)
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
		quotaInfoList             []types.QuotaInfo
		unconfirmedList           []*ledger.AccountBlock
		quotaTotal, quotaAddition uint64
		err                       error
		name                      string
	}{
		{types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			0, 0, nil, "no_quota",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(1),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			21000, 21000, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge1",
		},
		{types.Address{}, big.NewInt(20007), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			42000, 0, nil, "get_quota_by_pledge2",
		},
		{types.Address{}, big.NewInt(30033), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 42001, QuotaUsedTotal: 42001},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			41998, 0, nil, "get_quota_by_pledge3",
		},
		{types.Address{}, big.NewInt(10001), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge4",
		},
		{types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 21000, nil, "get_quota_by_difficulty1",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			42000, 21000, nil, "get_quota_by_difficulty2",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			21000, 21000, nil, "get_quota_by_difficulty3",
		},
		{types.Address{}, big.NewInt(10), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			0, 0, nil, "quota_total_less_than_used1",
		},
		{types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 40000, QuotaUsedTotal: 40000},
			}, []*ledger.AccountBlock{{Quota: 20000}},
			22000, 21000, nil, "quota_total_less_than_used2",
		},
		{types.Address{}, big.NewInt(1197189), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			}, []*ledger.AccountBlock{},
			1000000, 0, nil, "block_quota_limit_reached1",
		},
		{types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached2",
		},
		{types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached3",
		},
	}
	InitQuotaConfig(false, false)
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, updateUnconfirmedQuotaInfo(testCase.quotaInfoList, testCase.unconfirmedList), testCase.unconfirmedList}
		quotaTotal, quotaAddition, err := CalcQuotaForBlock(db, testCase.addr, getPledgeAmount(testCase.pledgeAmount), testCase.difficulty)
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
			nodeConfig.difficultyList[index/75],
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
			nodeConfig.difficultyList[index/75],
		)
		index += 75
	}
}

func TestPrintQuota(t *testing.T) {
	InitQuotaConfig(false, false)
	for i := 75; i < len(nodeConfig.sectionList); i = i + 75 {
		pledgeAmount := new(big.Int).Quo(nodeConfig.pledgeAmountList[i], util.AttovPerVite)
		fmt.Printf("| $(%v, %v]$ | %v | %v | %v | %v |\n", nodeConfig.sectionList[i-75], nodeConfig.sectionList[i], uint64(i)*quotaForSection, i/75, pledgeAmount, nodeConfig.difficultyList[i])
	}
}
