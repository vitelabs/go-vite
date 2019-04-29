package quota

import (
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
	calcQuotaFunc    func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, quotaUnconfirmed, quotaAvg uint64, err error)
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
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, quotaUnconfirmed, quotaAvg uint64, err error) {
			// TODO update quotaUnconfirmed for chain
			return 1000000, 1000000, 0, 0, 0, nil
		}
	} else {
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, quotaUnconfirmed, quotaAvg uint64, err error) {
			return calcQuotaV3(db, addr, pledgeAmount, difficulty)
		}
	}
}

type quotaDb interface {
	GetQuotaUsedList(address types.Address) []types.QuotaInfo
	GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock
}

func CalcBlockQuota(db quotaDb, block *ledger.AccountBlock) (uint64, error) {
	if !util.IsPoW(block) {
		return block.QuotaUsed, nil
	}
	powQuota, err := calcPoWQuota(db, block.Difficulty)
	if err != nil {
		return 0, err
	}
	if block.QuotaUsed > powQuota {
		return block.QuotaUsed - powQuota, nil
	}
	return 0, nil
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (types.Quota, error) {
	quotaTotal, pledgeQuota, _, quotaUnconfirmed, quotaAvg, err := nodeConfig.calcQuotaFunc(db, beneficial, pledgeAmount, big.NewInt(0))
	return types.NewQuota(pledgeQuota, quotaTotal, quotaUnconfirmed, quotaAvg), err
}

func CalcQuotaForBlock(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition uint64, err error) {
	quotaTotal, _, quotaAddition, quotaUsed, _, err := nodeConfig.calcQuotaFunc(db, addr, pledgeAmount, difficulty)
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
func CheckQuota(db quotaDb, q types.Quota, addr types.Address) bool {
	if unconfirmedBlocks := db.GetUnconfirmedBlocks(addr); len(unconfirmedBlocks) > 0 &&
		unconfirmedBlocks[len(unconfirmedBlocks)-1].BlockType == ledger.BlockTypeReceiveError {
		return false
	}
	if q.Current() >= q.Avg() {
		return true
	} else {
		return false
	}
}

func calcQuotaV3(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, quotaUnconfirmed, quotaAvg uint64, err error) {
	powFlag := difficulty != nil && difficulty.Sign() > 0
	if powFlag {
		canPoW, err := CanPoW(db, addr)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		if !canPoW {
			return 0, 0, 0, 0, 0, util.ErrCalcPoWTwice
		}
	}
	return calcQuotaTotal(db, pledgeAmount, difficulty)
}
func calcQuotaTotal(db quotaDb, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, quotaUnconfirmed, quotaAvg uint64, err error) {
	quotaPledge, err = calcPledgeQuota(db, pledgeAmount)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	quotaAddition, err = calcPoWQuota(db, difficulty)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	// TODO get quota Used
	return 0, quotaPledge, quotaAddition, 0, 0, nil
}

func calcPledgeQuota(db quotaDb, pledgeAmount *big.Int) (uint64, error) {
	// TODO update param by quotaUsed during past 74 snapshot blocks
	// TODO detailing calc result
	if pledgeAmount == nil || pledgeAmount.Sign() <= 0 {
		return 0, nil
	}
	return calcQuotaByIndex(getIndexInBigIntList(pledgeAmount, nodeConfig.pledgeAmountList, 0, len(nodeConfig.sectionList)-1)), nil
}

func calcPoWQuota(db quotaDb, difficulty *big.Int) (uint64, error) {
	// TODO update param by quotaUsed during past 74 snapshot blocks
	// TODO detailing calc result
	if difficulty == nil || difficulty.Sign() <= 0 {
		return 0, nil
	}
	return calcQuotaByIndex(getIndexInBigIntList(difficulty, nodeConfig.difficultyList, 0, len(nodeConfig.sectionList)-1)), nil
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

func CanPoW(db quotaDb, address types.Address) (bool, error) {
	blocks := db.GetUnconfirmedBlocks(address)
	for _, b := range blocks {
		if util.IsPoW(b) {
			return false, nil
		}
	}
	return true, nil
}

func CalcPoWDifficulty(quotaRequired uint64, q types.Quota, pledgeAmount *big.Int) (*big.Int, error) {
	if quotaRequired > quotaLimitForBlock {
		return nil, util.ErrBlockQuotaLimitReached
	}
	if q.Current() >= quotaRequired {
		return big.NewInt(0), nil
	}

	index := getIndexByQuota(quotaRequired)
	difficulty := nodeConfig.difficultyList[index]
	return difficulty, nil
}
