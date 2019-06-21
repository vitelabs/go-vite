package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var (
	testTokenId = types.TokenTypeId{'T', 'E', 'S', 'T', ' ', 'T', 'O', 'K', 'E', 'N'}
)

type RunV2TestCase struct {
	caseType string
	// send create params
	code         []byte
	contractType uint8
	confirmTimes uint8
	seedCount    uint8
	quotaRatio   uint8
	gid          types.Gid
	// latest snapshot block
	snapshotHeight uint64
	// block info
	data    []byte
	amount  *big.Int
	tokenId *types.TokenTypeId
	// quota
	pledgeBeneficialAmount *big.Int
	// balance and storage
	balanceMap map[types.TokenTypeId]*big.Int
	// return data
	err         error
	isRetry     bool
	returnBlock *ledger.AccountBlock
}

func TestVM_RunSendCreate(t *testing.T) {
	testCases := []RunV2TestCase{
		{
			caseType:               "quota_not_enough",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           2,
			seedCount:              2,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: big.NewInt(0),
			err:                    util.ErrOutOfQuota,
		},
		{
			caseType:               "contract_type_error_before_hardfork",
			contractType:           0,
			confirmTimes:           2,
			seedCount:              2,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         1,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "confirm_time_error_before_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           76,
			seedCount:              2,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         1,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidConfirmTime,
		},
		{
			caseType:               "confirm_time_error_with_code_before_hardfork",
			code:                   []byte{0x43},
			contractType:           util.SolidityPPContractType,
			confirmTimes:           0,
			seedCount:              0,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         1,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidConfirmTime,
		},
		{
			caseType:               "quota_ratio_error_before_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             3,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         1,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidQuotaRatio,
		},
		{
			caseType:               "gid_error_before_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.SNAPSHOT_GID,
			snapshotHeight:         1,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "data_length_error_before_hardfork",
			snapshotHeight:         1,
			data:                   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2},
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "contract_type_error_after_hardfork",
			contractType:           0,
			confirmTimes:           2,
			seedCount:              2,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "confirm_time_error_after_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           76,
			seedCount:              2,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidConfirmTime,
		},
		{
			caseType:               "confirm_time_error_with_code_after_hardfork",
			code:                   []byte{0x43},
			contractType:           util.SolidityPPContractType,
			confirmTimes:           0,
			seedCount:              0,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidConfirmTime,
		},
		{
			caseType:               "seed_count_error_after_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              5,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidSeedCount,
		},
		{
			caseType:               "seed_count_with_code_error_after_hardfork",
			code:                   []byte{0x4a},
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              0,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidSeedCount,
		},
		{
			caseType:               "quota_ratio_error_after_hardfork",
			code:                   []byte{},
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             3,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidQuotaRatio,
		},
		{
			caseType:               "gid_error_after_hardfork",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.SNAPSHOT_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "data_length_error_after_hardfork",
			snapshotHeight:         100,
			data:                   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3},
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInvalidMethodParam,
		},
		{
			caseType:               "insufficient_balance",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInsufficientBalance,
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: big.NewInt(1),
			},
		},
		{
			caseType:               "insufficient_balance_with_amount",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInsufficientBalance,
			amount:                 big.NewInt(1),
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)),
			},
		},
		{
			caseType:               "insufficient_balance_with_different_token",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInsufficientBalance,
			amount:                 big.NewInt(1),
			tokenId:                &testTokenId,
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)),
			},
		},
		{
			caseType:               "insufficient_balance_with_different_token_2",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInsufficientBalance,
			amount:                 big.NewInt(2),
			tokenId:                &testTokenId,
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)),
				testTokenId:        big.NewInt(1),
			},
		},
		{
			caseType:               "insufficient_balance_with_different_token_3",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			err:                    util.ErrInsufficientBalance,
			amount:                 big.NewInt(1),
			tokenId:                &testTokenId,
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: new(big.Int).Mul(big.NewInt(9), big.NewInt(1e18)),
				testTokenId:        big.NewInt(1),
			},
		},
		{
			caseType:               "normal",
			contractType:           util.SolidityPPContractType,
			confirmTimes:           4,
			seedCount:              1,
			quotaRatio:             20,
			gid:                    types.DELEGATE_GID,
			snapshotHeight:         100,
			pledgeBeneficialAmount: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)),
			amount:                 big.NewInt(10),
			balanceMap: map[types.TokenTypeId]*big.Int{
				ledger.ViteTokenId: new(big.Int).Mul(big.NewInt(11), big.NewInt(1e18)),
			},
			returnBlock: &ledger.AccountBlock{
				Quota:     21952,
				QuotaUsed: 21952,
				Fee:       new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)),
			},
		},
	}
	runTest(testCases, t)
}

func runTest(testCases []RunV2TestCase, t *testing.T) {
	vm := NewVM(nil)
	addr, _ := types.HexToAddress("vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a")
	prevHash, _ := types.HexToHash("82a8ecfe0df3dea6256651ee3130747386d4d6ab61201ce0050a6fe394a0f595")
	latestHeight := uint64(1)
	latestSnapshotHash, _ := types.HexToHash("41257eb3d17c4cd860a704a9f3aada83a479a3e634879049892587c009be93a3")

	for _, testCase := range testCases {
		var sendCreateData []byte
		if len(testCase.data) > 0 {
			sendCreateData = testCase.data
		} else {
			sendCreateData = util.GetCreateContractData(
				testCase.code,
				testCase.contractType,
				testCase.confirmTimes,
				testCase.seedCount,
				testCase.quotaRatio,
				testCase.gid,
				testCase.snapshotHeight)
		}

		prevAccountBlock := &ledger.AccountBlock{
			BlockType:      ledger.BlockTypeReceive,
			Height:         latestHeight,
			Hash:           prevHash,
			PrevHash:       types.ZERO_HASH,
			AccountAddress: addr,
		}
		tokenId := ledger.ViteTokenId
		if testCase.tokenId != nil {
			tokenId = *testCase.tokenId
		}
		amount := big.NewInt(0)
		if testCase.amount != nil {
			amount = testCase.amount
		}
		sendCreateBlock := &ledger.AccountBlock{
			BlockType:      ledger.BlockTypeSendCreate,
			PrevHash:       prevHash,
			Height:         latestHeight + 1,
			AccountAddress: addr,
			Amount:         amount,
			TokenId:        tokenId,
			Data:           sendCreateData,
		}

		currentTime := time.Now()
		latestSnapshotBlock := &ledger.SnapshotBlock{
			Hash:      latestSnapshotHash,
			PrevHash:  types.ZERO_HASH,
			Height:    testCase.snapshotHeight,
			Timestamp: &currentTime,
		}

		quotaInfoList := make([]types.QuotaInfo, 0, 75)
		for i := 0; i < 75; i++ {
			quotaInfoList = append(quotaInfoList, types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0})
		}

		db := NewMockDB(&addr, latestSnapshotBlock, prevAccountBlock, quotaInfoList, testCase.pledgeBeneficialAmount, testCase.balanceMap)

		vmBlock, isRetry, err := vm.RunV2(db, sendCreateBlock, nil, nil)
		if !errorEquals(err, testCase.err) {
			t.Fatalf("name: %v, error not match, expected %v, but got %v", testCase.caseType, testCase.err, err)
		}
		if isRetry != testCase.isRetry {
			t.Fatalf("name: %v, isRetry not match, expected %v, but got %v", testCase.caseType, testCase.isRetry, isRetry)
		}
		vmBlockNotNil := vmBlock != nil && vmBlock.AccountBlock != nil
		expectedToAddr := util.NewContractAddress(sendCreateBlock.AccountAddress, sendCreateBlock.Height, sendCreateBlock.PrevHash)
		if vmBlockNotNil && testCase.returnBlock == nil ||
			!vmBlockNotNil && testCase.returnBlock != nil {
			t.Fatalf("name: %v, return block not match, expected nil %v, but got nil %v", testCase.caseType, testCase.returnBlock == nil, !vmBlockNotNil)
		} else if vmBlockNotNil && testCase.returnBlock != nil {
			if vmBlock.AccountBlock.Fee.Cmp(testCase.returnBlock.Fee) != 0 {
				t.Fatalf("name: %v, fee not match, expected %v, but %v", testCase.caseType, testCase.returnBlock.Fee, vmBlock.AccountBlock.Fee)
			}
			if vmBlock.AccountBlock.Quota != testCase.returnBlock.Quota {
				t.Fatalf("name: %v, quota not match, expected %v, but %v", testCase.caseType, testCase.returnBlock.Quota, vmBlock.AccountBlock.Quota)
			}
			if vmBlock.AccountBlock.QuotaUsed != testCase.returnBlock.QuotaUsed {
				t.Fatalf("name: %v, quota used not match, expected %v, but %v", testCase.caseType, testCase.returnBlock.QuotaUsed, vmBlock.AccountBlock.QuotaUsed)
			}
			if vmBlock.AccountBlock.ToAddress != expectedToAddr {
				t.Fatalf("name: %v, to address not match, expected %v, but %v", testCase.caseType, expectedToAddr, vmBlock.AccountBlock.ToAddress)
			}
		}
		if vmBlockNotNil {
			if amount.Sign() > 0 {
				testCase.balanceMap[tokenId].Sub(testCase.balanceMap[tokenId], amount)
			}
			testCase.balanceMap[ledger.ViteTokenId].Sub(testCase.balanceMap[ledger.ViteTokenId], testCase.returnBlock.Fee)
			if checkBalanceMapResult := checkBalanceMap(db.balanceMap, testCase.balanceMap); len(checkBalanceMapResult) > 0 {
				t.Fatalf("name: %v, %v", testCase.caseType, checkBalanceMapResult)
			}
			got, _ := db.GetContractMetaInSnapshot(expectedToAddr, nil)
			if got == nil {
				t.Fatalf("name: %v, contract meta is nil", testCase.caseType)
			}
			if len(testCase.data) == 0 && (got.QuotaRatio != testCase.quotaRatio ||
				got.SendConfirmedTimes != testCase.confirmTimes ||
				got.Gid != testCase.gid ||
				(fork.IsSeedFork(testCase.snapshotHeight) && got.SeedConfirmedTimes != testCase.seedCount)) {
				t.Fatalf("name: %v, contract meta not match, expected [%v,%v,%v,%v], got [%v,%v,%v,%v]",
					testCase.caseType,
					got.QuotaRatio, got.SendConfirmedTimes, got.Gid, got.SeedConfirmedTimes,
					testCase.quotaRatio, testCase.confirmTimes, testCase.gid, testCase.seedCount)
			}
			if len(testCase.data) > 0 &&
				(got.QuotaRatio != util.GetQuotaRatioFromCreateContractData(testCase.data, testCase.snapshotHeight) ||
					got.SendConfirmedTimes != util.GetConfirmTimeFromCreateContractData(testCase.data) ||
					got.Gid != util.GetGidFromCreateContractData(testCase.data) ||
					(fork.IsSeedFork(testCase.snapshotHeight) && got.SeedConfirmedTimes != util.GetSeedCountFromCreateContractData(testCase.data))) {
				t.Fatalf("name: %v, contract meta not match, expected [%v,%v,%v,%v], got [%v]",
					testCase.caseType,
					got.QuotaRatio, got.SendConfirmedTimes, got.Gid, got.SeedConfirmedTimes,
					hex.EncodeToString(testCase.data))
			}
		}
	}
}

func errorEquals(expected, got error) bool {
	if (expected == nil && got != nil) || (expected != nil && got == nil) ||
		(expected != nil && got != nil && expected.Error() != got.Error()) {
		return false
	}
	return true
}

func checkBalanceMap(got map[types.TokenTypeId]*big.Int, expected map[types.TokenTypeId]*big.Int) string {
	gotCount := 0
	for _, v := range got {
		if v.Sign() > 0 {
			gotCount = gotCount + 1
		}
	}
	expectedCount := 0
	for _, v := range expected {
		if v.Sign() > 0 {
			expectedCount = expectedCount + 1
		}
	}
	if expectedCount != gotCount {
		return "expected len " + strconv.Itoa(expectedCount) + ", got len" + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		if v.Sign() == 0 {
			continue
		}
		if expectedV := expected[k]; v.Cmp(expectedV) != 0 {
			return k.String() + " token balance, expect " + expectedV.String() + ", got " + v.String()
		}
	}
	return ""
}
