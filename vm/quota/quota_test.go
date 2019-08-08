package quota

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
	"testing"
)

func initForkPointsForQuotaTest() {
	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{Height: 100, Version: 1},
		DexFork:  &config.ForkPoint{Height: 200, Version: 1},
		NewFork:  &config.ForkPoint{Height: 200, Version: 1}})
}

type testQuotaDb struct {
	addr                 types.Address
	quotaList            []types.QuotaInfo
	unconfirmedBlockList []*ledger.AccountBlock
	globalQuota          types.QuotaInfo
}

func (db *testQuotaDb) Address() *types.Address {
	return &db.addr
}
func (db *testQuotaDb) GetGlobalQuota() types.QuotaInfo {
	return db.globalQuota
}
func (db *testQuotaDb) GetQuotaUsedList(address types.Address) []types.QuotaInfo {
	return db.quotaList
}
func (db *testQuotaDb) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	return db.unconfirmedBlockList
}
func (db *testQuotaDb) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return 0, nil
}
func (db *testQuotaDb) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if len(db.unconfirmedBlockList) > 0 {
		return db.unconfirmedBlockList[len(db.unconfirmedBlockList)-1], nil
	} else {
		return nil, nil
	}
}

func TestCalcPoWDifficulty(t *testing.T) {
	InitQuotaConfig(false, false)
	initForkPointsForQuotaTest()
	testCases := []struct {
		sbHeight      uint64
		globalTotal   uint64
		quotaRequired uint64
		q             types.Quota
		difficulty    *big.Int
		err           error
		name          string
	}{
		{1, 0, 1000001, types.NewQuota(0, 0, 0, 0, false), nil, errors.New("quota limit for block reached"), "block_quota_limit_reached_before_hardfork"},
		{1, 0, 21000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(67108863), nil, "no_pledge_quota_before_hardfork"},
		{1, 0, 22000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(70689140), nil, "pledge_quota_not_enough_before_hardfork"},
		{1, 0, 21000, types.NewQuota(0, 21000, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_before_hardfork"},
		{1, 0, 21000, types.NewQuota(0, 21001, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_2_before_hardfork"},
		{200, 0, 1000001, types.NewQuota(0, 0, 0, 0, false), nil, errors.New("quota limit for block reached"), "block_quota_limit_reached_after_hardfork"},
		{200, 0, 21000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(67108863), nil, "no_pledge_quota_after_hardfork"},
		{200, 0, 22000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(70689140), nil, "pledge_quota_not_enough_after_hardfork"},
		{200, 0, 21000, types.NewQuota(0, 21000, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_after_hardfork"},
		{200, 0, 21000, types.NewQuota(0, 21001, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_2_after_hardfork"},
		{200, 0, 1000001, types.NewQuota(0, 0, 0, 0, false), nil, errors.New("quota limit for block reached"), "block_quota_limit_reached_with_congestion_after_hardfork"},
		{200, 74 * 51 * 21000, 21000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(67987247), nil, "no_pledge_quota_with_congestion_after_hardfork"},
		{200, 74 * 51 * 21000, 22000, types.NewQuota(0, 0, 0, 0, false), big.NewInt(71614386), nil, "pledge_quota_not_enough_with_congestion_after_hardfork"},
		{200, 74 * 51 * 21000, 21000, types.NewQuota(0, 21000, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_with_congestion_after_hardfork"},
		{200, 74 * 51 * 21000, 21000, types.NewQuota(0, 21001, 0, 0, false), big.NewInt(0), nil, "current_quota_enough_2_with_congestion_after_hardfork"},
	}
	for _, testCase := range testCases {
		difficulty, err := CalcPoWDifficulty(&testQuotaDb{globalQuota: types.QuotaInfo{QuotaUsedTotal: testCase.globalTotal}}, testCase.quotaRequired, testCase.q, testCase.sbHeight)
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
		db := &testQuotaDb{addr, nil, testCase.blockList, types.QuotaInfo{}}
		result := CanPoW(db, addr)
		if result != testCase.result {
			t.Fatalf("%v CanPoW failed, result not match, expected %v, got %v", testCase.name, testCase.result, result)
		}
	}
}

func TestCalcQuotaV3(t *testing.T) {
	testCases := []struct {
		sbHeight                                                               uint64
		globalQuota                                                            uint64
		addr                                                                   types.Address
		pledgeAmount                                                           *big.Int
		difficulty                                                             *big.Int
		quotaInfoList                                                          []types.QuotaInfo
		unconfirmedList                                                        []*ledger.AccountBlock
		quotaTotal, pledgeQuota, quotaAddition, snapshotCurrentQuota, quotaAvg uint64
		err                                                                    error
		name                                                                   string
	}{
		{1, 0,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			0, 0, 0, 0, 0, nil, "no_quota_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 21000, 0, 21000, 0, nil, "new_pledge_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, 21000, 0, 0, nil, "new_pow_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			210000, 21000, 0, 210000, 0, nil, "pledge_1_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			199500, 21000, 0, 210000, 10500, nil, "pledge_2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			136500, 21000, 0, 210000, 36750, nil, "pledge_3_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			136500, 21000, 0, 136500, 36750, nil, "pledge_4_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			31500, 21000, 0, 136500, 59500, nil, "pledge_5_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			31500, 21000, 0, 31500, 59500, nil, "pledge_6_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			0, 21000, 0, 31500, 57125, util.ErrInvalidUnconfirmedQuota, "pledge_7_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			0, 21000, 0, 31500, 52500, nil, "pledge_8_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			21000, 21000, 0, 21000, 52500, nil, "pledge_9_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			42000, 21000, 0, 42000, 52500, nil, "pledge_10_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			63000, 21000, 21000, 42000, 52500, nil, "pledge_and_pow_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			0, 0, 0, 0, 0, util.ErrCalcPoWTwice, "can_not_pow_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			31500, 21000, 21000, 42000, 48300, nil, "pledge_and_pow_2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			41000, 21000, 21000, 51500, 48300, nil, "calc_quota_used_before_hardfork",
		},

		{200, 0,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			0, 0, 0, 0, 0, nil, "no_quota_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 21000, 0, 21000, 0, nil, "new_pledge_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, 21000, 0, 0, nil, "new_pow_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			210000, 21000, 0, 210000, 0, nil, "pledge_1_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			199500, 21000, 0, 210000, 10500, nil, "pledge_2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			136500, 21000, 0, 210000, 36750, nil, "pledge_3_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			136500, 21000, 0, 136500, 36750, nil, "pledge_4_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			31500, 21000, 0, 136500, 59500, nil, "pledge_5_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			31500, 21000, 0, 31500, 59500, nil, "pledge_6_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			0, 21000, 0, 31500, 57125, util.ErrInvalidUnconfirmedQuota, "pledge_7_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			0, 21000, 0, 31500, 52500, nil, "pledge_8_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			21000, 21000, 0, 21000, 52500, nil, "pledge_9_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
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
			42000, 21000, 0, 42000, 52500, nil, "pledge_10_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			63000, 21000, 21000, 42000, 52500, nil, "pledge_and_pow_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			0, 0, 0, 0, 0, util.ErrCalcPoWTwice, "can_not_pow_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			31500, 21000, 21000, 42000, 48300, nil, "pledge_and_pow_2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
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
			41000, 21000, 21000, 51500, 48300, nil, "calc_quota_used_after_hardfork",
		},

		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			0, 0, 0, 0, 0, nil, "no_quota_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 21000, 0, 21000, 0, nil, "new_pledge_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(0), big.NewInt(67987247),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, 21000, 0, 0, nil, "new_pow_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			210000, 21000, 0, 210000, 0, nil, "pledge_1_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			199500, 21000, 0, 210000, 10500, nil, "pledge_2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			136500, 21000, 0, 210000, 36750, nil, "pledge_3_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			136500, 21000, 0, 136500, 36750, nil, "pledge_4_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			31500, 21000, 0, 136500, 59500, nil, "pledge_5_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			31500, 21000, 0, 31500, 59500, nil, "pledge_6_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			0, 21000, 0, 31500, 57125, util.ErrInvalidUnconfirmedQuota, "pledge_7_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			0, 21000, 0, 31500, 52500, nil, "pledge_8_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			21000, 21000, 0, 21000, 52500, nil, "pledge_9_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
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
			42000, 21000, 0, 42000, 52500, nil, "pledge_10_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
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
			63000, 21000, 21000, 42000, 52500, nil, "pledge_and_pow_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
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
			0, 0, 0, 0, 0, util.ErrCalcPoWTwice, "can_not_pow_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
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
			31500, 21000, 21000, 42000, 48300, nil, "pledge_and_pow_2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
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
			41000, 21000, 21000, 51500, 48300, nil, "calc_quota_used_with_congestion_after_hardfork",
		},
	}
	InitQuotaConfig(false, false)
	initForkPointsForQuotaTest()
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, updateUnconfirmedQuotaInfo(testCase.quotaInfoList, testCase.unconfirmedList), testCase.unconfirmedList, types.QuotaInfo{QuotaUsedTotal: testCase.globalQuota}}
		quotaTotal, pledgeQuota, quotaAddition, snapshotCurrentQuota, quotaAvg, _, err := calcQuotaV3(db, testCase.addr, getPledgeAmount(testCase.pledgeAmount), testCase.difficulty, testCase.sbHeight)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v calcQuotaV3 failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (quotaTotal != testCase.quotaTotal || pledgeQuota != testCase.pledgeQuota || quotaAddition != testCase.quotaAddition || snapshotCurrentQuota != testCase.snapshotCurrentQuota || quotaAvg != testCase.quotaAvg) {
			t.Fatalf("%v calcQuotaV3 failed, quota not match, expected (%v,%v,%v,%v,%v), got (%v,%v,%v,%v,%v)", testCase.name, testCase.quotaTotal, testCase.pledgeQuota, testCase.quotaAddition, testCase.snapshotCurrentQuota, testCase.quotaAvg, quotaTotal, pledgeQuota, quotaAddition, snapshotCurrentQuota, quotaAvg)
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
	db := &testQuotaDb{addr, updateUnconfirmedQuotaInfo(quotaInfoList, unConfirmedList), unConfirmedList, types.QuotaInfo{}}
	pledgeAmount := big.NewInt(10000)
	difficulty := big.NewInt(67108863)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calcQuotaV3(db, addr, pledgeAmount, difficulty, 1)
	}
}

func TestCalcQuotaForBlock(t *testing.T) {
	testCases := []struct {
		sbHeight                  uint64
		globalQuota               uint64
		addr                      types.Address
		pledgeAmount              *big.Int
		difficulty                *big.Int
		quotaInfoList             []types.QuotaInfo
		unconfirmedList           []*ledger.AccountBlock
		quotaTotal, quotaAddition uint64
		err                       error
		name                      string
	}{
		{1, 0,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			0, 0, nil, "no_quota_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(0), big.NewInt(1),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			21000, 21000, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge1_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(20007), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			42000, 0, nil, "get_quota_by_pledge2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(30033), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 42001, QuotaUsedTotal: 42001},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			41998, 0, nil, "get_quota_by_pledge3_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10001), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge4_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 21000, nil, "get_quota_by_difficulty1_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			42000, 21000, nil, "get_quota_by_difficulty2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			21000, 21000, nil, "get_quota_by_difficulty3_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			0, 0, nil, "quota_total_less_than_used1_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 40000, QuotaUsedTotal: 40000},
			}, []*ledger.AccountBlock{{Quota: 20000}},
			22000, 21000, nil, "quota_total_less_than_used2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			}, []*ledger.AccountBlock{},
			1000000, 0, nil, "block_quota_limit_reached1_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached2_before_hardfork",
		},
		{1, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached3_before_hardfork",
		},

		{200, 0,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			0, 0, nil, "no_quota_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(0), big.NewInt(1),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			21000, 21000, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge1_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(20007), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			42000, 0, nil, "get_quota_by_pledge2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(30033), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 42001, QuotaUsedTotal: 42001},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			41998, 0, nil, "get_quota_by_pledge3_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10001), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge4_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(0), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 21000, nil, "get_quota_by_difficulty1_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			42000, 21000, nil, "get_quota_by_difficulty2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			21000, 21000, nil, "get_quota_by_difficulty3_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			0, 0, nil, "quota_total_less_than_used1_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(10000), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 40000, QuotaUsedTotal: 40000},
			}, []*ledger.AccountBlock{{Quota: 20000}},
			22000, 21000, nil, "quota_total_less_than_used2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			}, []*ledger.AccountBlock{},
			1000000, 0, nil, "block_quota_limit_reached1_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached2_after_hardfork",
		},
		{200, 0,
			types.Address{}, big.NewInt(1197189), big.NewInt(67108863),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached3_after_hardfork",
		},

		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(0), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			0, 0, nil, "no_quota_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(0), big.NewInt(1),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{{Nonce: []byte{1}}},
			0, 0, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 0, Nonce: []byte{1}}},
			21000, 21000, errors.New("calc PoW twice referring to one snapshot block"), "cannot_pow2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(0),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge1_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(20269), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			42000, 0, nil, "get_quota_by_pledge2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(30427), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 2, QuotaTotal: 42001, QuotaUsedTotal: 42001},
			},
			[]*ledger.AccountBlock{{Quota: 21000}, {Quota: 21001}},
			41998, 0, nil, "get_quota_by_pledge3_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10132), big.NewInt(0),
			[]types.QuotaInfo{},
			[]*ledger.AccountBlock{},
			21000, 0, nil, "get_quota_by_pledge4_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(0), big.NewInt(67987247),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			21000, 21000, nil, "get_quota_by_difficulty1_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			42000, 21000, nil, "get_quota_by_difficulty2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{{Quota: 21000}},
			21000, 21000, nil, "get_quota_by_difficulty3_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(11), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			0, 0, nil, "quota_total_less_than_used1_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(10131), big.NewInt(67987247),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 40000, QuotaUsedTotal: 40000},
			}, []*ledger.AccountBlock{{Quota: 20000}},
			22000, 21000, nil, "quota_total_less_than_used2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(1212859), big.NewInt(0),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			}, []*ledger.AccountBlock{},
			1000000, 0, nil, "block_quota_limit_reached1_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(1212859), big.NewInt(67987247),
			[]types.QuotaInfo{
				{BlockCount: 1, QuotaTotal: 21000, QuotaUsedTotal: 21000},
			},
			[]*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached2_with_congestion_after_hardfork",
		},
		{200, 74 * 51 * 21000,
			types.Address{}, big.NewInt(1212859), big.NewInt(67987247),
			[]types.QuotaInfo{}, []*ledger.AccountBlock{},
			1000000, 21000, nil, "block_quota_limit_reached3_with_congestion_after_hardfork",
		},
	}
	InitQuotaConfig(false, false)
	initForkPointsForQuotaTest()
	for _, testCase := range testCases {
		db := &testQuotaDb{testCase.addr, updateUnconfirmedQuotaInfo(testCase.quotaInfoList, testCase.unconfirmedList), testCase.unconfirmedList, types.QuotaInfo{QuotaUsedTotal: testCase.globalQuota}}
		quotaTotal, quotaAddition, err := CalcQuotaForBlock(db, testCase.addr, getPledgeAmount(testCase.pledgeAmount), testCase.difficulty, testCase.sbHeight)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v TestCalcQuotaForBlock failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (quotaTotal != testCase.quotaTotal || quotaAddition != testCase.quotaAddition) {
			t.Fatalf("%v TestCalcQuotaForBlock failed, quota not match, expected (%v,%v), got (%v,%v)", testCase.name, testCase.quotaTotal, testCase.quotaAddition, quotaTotal, quotaAddition)
		}
	}
}

func TestCalcPledgeAmountByUtps(t *testing.T) {
	InitQuotaConfig(false, false)
	testCases := []struct {
		utps           float64
		expectedResult *big.Int
		err            error
	}{
		{
			-1,
			nil,
			util.ErrInvalidMethodParam,
		},
		{
			0,
			big.NewInt(0),
			nil,
		},
		{
			0.013,
			new(big.Int).Mul(big.NewInt(134), big.NewInt(1e18)),
			nil,
		},
		{
			0.026,
			new(big.Int).Mul(big.NewInt(267), big.NewInt(1e18)),
			nil,
		},
		{
			1.0,
			new(big.Int).Mul(big.NewInt(1e4), big.NewInt(1e18)),
			nil,
		},
		{
			48,
			nil,
			util.ErrInvalidMethodParam,
		},
	}
	for _, utps := range testCases {
		result, error := CalcPledgeAmountByUtps(utps.utps)
		if (error == nil && utps.err != nil) || (error != nil && utps.err == nil) ||
			(error != nil && utps.err != nil && error.Error() != utps.err.Error()) {
			t.Fatalf("param: %v, error expected %v, but got %v", utps.utps, utps.err, error)
		}
		if error == nil && utps.err == nil && result.Cmp(utps.expectedResult) != 0 {
			t.Fatalf("param: %v, result expected %v, but got %v", utps.utps, utps.expectedResult, result)
		}
	}
}

var (
	testTokenId     = types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}
	testAddr, _     = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	testUint8       = uint8(1)
	testUint16      = uint16(1)
	testUint32      = uint32(1)
	testUint64      = uint64(1)
	testUint256     = big.NewInt(1)
	testInt32       = int32(1)
	testInt64       = int64(1)
	testBool        = true
	testTokenSymbol = "ABCDEFGHIJ"
	testPrice       = "1.1111111111111111111"
	testOrderId, _  = hex.DecodeString("01010101010101010101010101010101010101010101")
	numeratorList   = makeNumeratorList()
)

func makeNumeratorList() []uint64 {
	numeratorList := make([]uint64, 0)
	for i := uint64(1); i <= 75*20; i++ {
		if i <= 75 || i%75 == 0 {
			numeratorList = append(numeratorList, i)
		}
	}
	return numeratorList
}

func TestCalcDexQuota(t *testing.T) {
	InitQuotaConfig(false, false)
	gasTable := util.GasTableByHeight(1)
	dataLenMap := make(map[string]int)
	methodNameDexFundUserDepositData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserDeposit)
	dataLenMap[abi.MethodNameDexFundUserDeposit] = len(methodNameDexFundUserDepositData)
	methodNameDexFundUserWithdrawData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserWithdraw, testTokenId, testUint256)
	dataLenMap[abi.MethodNameDexFundUserWithdraw] = len(methodNameDexFundUserWithdrawData)
	methodNameDexFundNewOrderData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewOrder, testTokenId, testTokenId, testBool, testUint8, testPrice, testUint256)
	dataLenMap[abi.MethodNameDexFundNewOrder] = len(methodNameDexFundNewOrderData)
	methodNameDexFundPeriodJobData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPeriodJob, testUint64, testUint8)
	dataLenMap[abi.MethodNameDexFundPeriodJob] = len(methodNameDexFundPeriodJobData)
	methodNameDexFundNewMarketData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewMarket, testTokenId, testTokenId)
	dataLenMap[abi.MethodNameDexFundNewMarket] = len(methodNameDexFundNewMarketData)
	methodNameDexFundPledgeForVxData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPledgeForVx, testUint8, testUint256)
	dataLenMap[abi.MethodNameDexFundPledgeForVx] = len(methodNameDexFundPledgeForVxData)
	methodNameDexFundPledgeForVipData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundPledgeForVip, testUint8)
	dataLenMap[abi.MethodNameDexFundPledgeForVip] = len(methodNameDexFundPledgeForVipData)
	methodNameDexFundOwnerConfigData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundOwnerConfig, testUint8, testAddr, testAddr, testAddr, testBool, testAddr, testAddr)
	dataLenMap[abi.MethodNameDexFundOwnerConfig] = len(methodNameDexFundOwnerConfigData)
	methodNameDexFundOwnerConfigTradeData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundOwnerConfigTrade, testUint8, testTokenId, testTokenId, testBool, testTokenId, testUint8, testUint8, testUint256, testUint8, testUint256)
	dataLenMap[abi.MethodNameDexFundOwnerConfigTrade] = len(methodNameDexFundOwnerConfigTradeData)
	methodNameDexFundMarketOwnerConfigData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundMarketOwnerConfig, testUint8, testTokenId, testTokenId, testAddr, testInt32, testInt32, testBool)
	dataLenMap[abi.MethodNameDexFundMarketOwnerConfig] = len(methodNameDexFundMarketOwnerConfigData)
	methodNameDexFundTransferTokenOwnerData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundTransferTokenOwner, testTokenId, testAddr)
	dataLenMap[abi.MethodNameDexFundTransferTokenOwner] = len(methodNameDexFundTransferTokenOwnerData)
	methodNameDexFundNotifyTimeData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNotifyTime, testInt64)
	dataLenMap[abi.MethodNameDexFundNotifyTime] = len(methodNameDexFundNotifyTimeData)
	methodNameDexFundNewInviterData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewInviter)
	dataLenMap[abi.MethodNameDexFundNewInviter] = len(methodNameDexFundNewInviterData)
	methodNameDexFundBindInviteCodeData, _ := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundBindInviteCode, testUint32)
	dataLenMap[abi.MethodNameDexFundBindInviteCode] = len(methodNameDexFundBindInviteCodeData)
	methodNameDexTradeCancelOrderData, _ := abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, testOrderId)
	dataLenMap[abi.MethodNameDexTradeCancelOrder] = len(methodNameDexTradeCancelOrderData)
	fmt.Println("quota for dex tx")
	for name, length := range dataLenMap {
		f := float64(gasTable.TxGas+uint64(length)*gasTable.TxDataGas) / float64(gasTable.TxGas) / float64(75)
		minPledgeAmount, _ := CalcPledgeAmountByUtps(f)
		fmt.Printf("%v\t%v\t%v\n", name, gasTable.TxGas+uint64(length)*gasTable.TxDataGas, minPledgeAmount.Div(minPledgeAmount, big.NewInt(1e18)))
	}
	fmt.Println("pledge amount for dex tx")
	for name, length := range dataLenMap {
		for _, numerator := range numeratorList {
			f := float64(numerator*(gasTable.TxGas+uint64(length)*gasTable.TxDataGas)) / float64(gasTable.TxGas) / float64(75)
			pledgeAmount, _ := CalcPledgeAmountByUtps(f)
			if numerator < 75 {
				fmt.Printf("%v\t%v\t%v\n", name, strconv.Itoa(int(numerator))+"("+strconv.Itoa(int(numerator))+"/75 tps)", pledgeAmount.Div(pledgeAmount, big.NewInt(1e18)))
			} else {
				fmt.Printf("%v\t%v\t%v\n", name, strconv.Itoa(int(numerator))+"("+strconv.Itoa(int(numerator/75))+" tps)", pledgeAmount.Div(pledgeAmount, big.NewInt(1e18)))
			}
		}
	}
}
