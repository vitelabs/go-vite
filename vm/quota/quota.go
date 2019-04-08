package quota

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

type NodeConfig struct {
	QuotaParams
	sectionList      []*big.Float
	difficultyList   []*big.Int
	pledgeAmountList []*big.Int
	calcQuotaFunc    func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64, err error)
}

var nodeConfig NodeConfig

func InitQuotaConfig(isTest, isTestParam bool) {
	sectionList := make([]*big.Float, len(sectionStrList))
	for i, str := range sectionStrList {
		sectionList[i], _ = new(big.Float).SetPrec(precForFloat).SetString(str)
	}
	if isTestParam {
		nodeConfig = NodeConfig{
			QuotaParams:      QuotaParamTestnet,
			sectionList:      sectionList,
			difficultyList:   difficultyListTestnet,
			pledgeAmountList: pledgeAmountListTestnet}
	} else {
		nodeConfig = NodeConfig{
			QuotaParams:      QuotaParamMainnet,
			sectionList:      sectionList,
			difficultyList:   difficultyListMainnet,
			pledgeAmountList: pledgeAmountListMainnet}
	}
	if isTest {
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64, err error) {
			return 1000000, 0, 0, 0, nil
		}
	} else {
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64, err error) {
			return calcQuotaV3(db, addr, pledgeAmount, difficulty)
		}
	}
}

type quotaDb interface {
	GetQuotaUsed(address *types.Address) (quotaUsed uint64, blockCount uint64)
	GetUnconfirmedBlocks() []*ledger.AccountBlock
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (types.Quota, error) {
	quotaTotal, _, quotaUsed, quotaAvg, err := nodeConfig.calcQuotaFunc(db, beneficial, pledgeAmount, big.NewInt(0))
	return types.NewQuota(quotaTotal, quotaUsed, quotaAvg), err
}

func CalcQuotaForBlock(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition uint64, err error) {
	quotaTotal, quotaAddition, quotaUsed, _, err := nodeConfig.calcQuotaFunc(db, addr, pledgeAmount, difficulty)
	if err != nil {
		return 0, 0, err
	}
	if quotaTotal-quotaAddition <= quotaUsed {
		// pledge amount changed in past 75 snapshot blocks, use PoW quota for new block only
		if quotaAddition > 0 {
			if quotaAddition > quotaLimitForBlock {
				return quotaLimitForBlock, quotaLimitForBlock, nil
			}
			return quotaAddition, quotaAddition, nil
		}
		return 0, 0, nil
	}
	current := quotaTotal - quotaUsed
	if current > quotaLimitForBlock {
		if quotaAddition > quotaLimitForBlock {
			return quotaLimitForBlock, quotaLimitForBlock, nil
		} else {
			return quotaLimitForBlock, quotaAddition, nil
		}
	} else {
		return current, quotaAddition, nil
	}
}

func CalcCreateQuota(fee *big.Int) uint64 {
	return quotaForCreateContract
}

// Check whether current quota of a contract account is enough to receive a new block
func CheckQuota(db quotaDb, q types.Quota) bool {
	if unconfirmedBlocks := db.GetUnconfirmedBlocks(); len(unconfirmedBlocks) > 0 &&
		unconfirmedBlocks[len(unconfirmedBlocks)-1].BlockType == ledger.BlockTypeReceiveError {
		return false
	}
	if q.Current() >= q.Avg() {
		return true
	} else {
		return false
	}
}

func calcQuotaV3(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64, err error) {
	powFlag := difficulty != nil && difficulty.Sign() > 0
	if powFlag {
		canPoW, err := CanPoW(db)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if !canPoW {
			return 0, 0, 0, 0, util.ErrCalcPoWTwice
		}
	}
	quotaTotal = calcQuotaTotal(pledgeAmount, difficulty)
	if powFlag {
		quotaAddition = quotaTotal - calcQuotaTotal(pledgeAmount, helper.Big0)
	} else {
		quotaAddition = 0
	}
	quotaUsed, quotaAvg = calcQuotaUsed(db, addr)
	return quotaTotal, quotaAddition, quotaUsed, quotaAvg, nil
}
func calcQuotaTotal(pledgeAmount *big.Int, difficulty *big.Int) uint64 {
	if (pledgeAmount == nil || pledgeAmount.Sign() <= 0) && (difficulty == nil || difficulty.Sign() <= 0) {
		return 0
	} else if pledgeAmount == nil || pledgeAmount.Sign() <= 0 {
		return calcQuotaByIndex(getIndexInBigIntList(difficulty, nodeConfig.difficultyList, 0, len(nodeConfig.sectionList)-1))
	} else if difficulty == nil || difficulty.Sign() <= 0 {
		return calcQuotaByIndex(getIndexInBigIntList(pledgeAmount, nodeConfig.pledgeAmountList, 0, len(nodeConfig.sectionList)-1))
	} else {
		return calcQuotaByPledgeAmountAndDifficulty(pledgeAmount, difficulty)
	}
}

func calcQuotaByPledgeAmountAndDifficulty(pledgeAmount, difficulty *big.Int) uint64 {
	tmpFloat1 := new(big.Float).SetPrec(precForFloat).SetInt(pledgeAmount)
	tmpFloat1 = tmpFloat1.Mul(tmpFloat1, nodeConfig.paramA)
	tmpFloat2 := new(big.Float).SetPrec(precForFloat).SetInt(difficulty)
	tmpFloat2 = tmpFloat2.Mul(tmpFloat2, nodeConfig.paramB)
	tmpFloat1 = tmpFloat1.Add(tmpFloat1, tmpFloat2)
	return calcQuotaByIndex(getIndexInSection(tmpFloat1))
}

func calcQuotaUsed(db quotaDb, addr types.Address) (uint64, uint64) {
	quotaUsed, txNum := db.GetQuotaUsed(&addr)
	if txNum == 0 {
		return 0, 0
	} else {
		return quotaUsed, quotaUsed / uint64(txNum)
	}
}

func calcQuotaByIndex(index int) uint64 {
	if index >= 0 && index < len(nodeConfig.sectionList) {
		return uint64(index) * quotaForSection
	}
	return 0
}

func getIndexByQuota(q uint64) int {
	return int((q + quotaForSection - 1) / quotaForSection)
}

// Get the largest index
// which makes sectionList[index] <= x
func getIndexInSection(x *big.Float) int {
	return getIndexInFloatList(x, nodeConfig.sectionList, 0, len(nodeConfig.sectionList)-1)
}
func getIndexInFloatList(x *big.Float, list []*big.Float, left, right int) int {
	if left == right {
		if left == 0 || list[left].Cmp(x) <= 0 {
			return left
		} else {
			return left - 1
		}
	}
	mid := (left + right + 1) / 2
	cmp := nodeConfig.sectionList[mid].Cmp(x)
	if cmp == 0 {
		return mid
	} else if cmp > 0 {
		return getIndexInFloatList(x, list, left, mid-1)
	} else {
		return getIndexInFloatList(x, list, mid, right)
	}
}

func getIndexInBigIntList(x *big.Int, list []*big.Int, left, right int) int {
	if left == right {
		return left
	}
	mid := (left + right + 1) / 2
	cmp := list[mid].Cmp(x)
	if cmp == 0 {
		return mid
	} else if cmp > 0 {
		return getIndexInBigIntList(x, list, left, mid-1)
	} else {
		return getIndexInBigIntList(x, list, mid, right)
	}
}

func CanPoW(db quotaDb) (bool, error) {
	blocks := db.GetUnconfirmedBlocks()
	for _, b := range blocks {
		if util.IsPoW(b) {
			return false, nil
		}
	}
	return true, nil
}

func CalcPoWDifficulty(quotaRequired uint64, q types.Quota, pledgeAmount *big.Int) (*big.Int, error) {
	if quotaRequired > quotaLimitForBlock {
		return nil, errors.New("quota limit for block reached")
	}
	expectedTotal := quotaRequired + q.Used()
	if expectedTotal > quotaLimitForAccount {
		return nil, errors.New("quota limit for account reached")
	}
	if q.Total() >= expectedTotal {
		return big.NewInt(0), nil
	}
	tmpFloat := new(big.Float).SetPrec(precForFloat)
	if pledgeAmount.Sign() > 0 {
		tmpFloat.SetInt(pledgeAmount)
		tmpFloat = tmpFloat.Mul(tmpFloat, nodeConfig.paramA)
	} else {
		tmpFloat.SetInt64(0)
	}
	index := getIndexByQuota(expectedTotal)
	var difficulty *big.Int
	if tmpFloat.Cmp(nodeConfig.sectionList[index]) >= 0 {
		difficulty = big.NewInt(0)
	} else {
		tmpFloat = tmpFloat.Sub(nodeConfig.sectionList[index], tmpFloat)
		tmpFloat = tmpFloat.Quo(tmpFloat, nodeConfig.paramB)
		difficulty, _ = tmpFloat.Int(nil)
		difficulty = difficulty.Add(difficulty, helper.Big1)
	}
	for calcQuotaByPledgeAmountAndDifficulty(pledgeAmount, difficulty) < expectedTotal {
		difficulty.Add(difficulty, helper.Big1)
	}
	return difficulty, nil
}
