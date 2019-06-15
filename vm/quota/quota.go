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
	calcQuotaFunc    func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, err error)
}

var nodeConfig NodeConfig

func InitQuotaConfig(isTest, isTestParam bool) {
	sectionList := make([]*big.Float, len(sectionStrList))
	for i, str := range sectionStrList {
		sectionList[i], _ = new(big.Float).SetPrec(precForFloat).SetString(str)
	}
	if isTestParam {
		nodeConfig = NodeConfig{
			QuotaParams:    QuotaParamTestnet,
			sectionList:    sectionList,
			difficultyList: difficultyListTestnet}
		pledgeAmountList := make([]*big.Int, len(pledgeAmountListTestnet))
		for i, str := range pledgeAmountListTestnet {
			pledgeAmountList[i], _ = new(big.Int).SetString(str, 10)
		}
		nodeConfig.pledgeAmountList = pledgeAmountList
	} else {
		nodeConfig = NodeConfig{
			QuotaParams:    QuotaParamMainnet,
			sectionList:    sectionList,
			difficultyList: difficultyListMainnet}
		pledgeAmountList := make([]*big.Int, len(pledgeAmountListMainnet))
		for i, str := range pledgeAmountListMainnet {
			pledgeAmountList[i], _ = new(big.Int).SetString(str, 10)
		}
		nodeConfig.pledgeAmountList = pledgeAmountList
	}
	if isTest {
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, err error) {
			return 75000000, 1000000, 0, 1000000, 0, false, nil
		}
	} else {
		nodeConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, err error) {
			return calcQuotaV3(db, addr, pledgeAmount, difficulty)
		}
	}
}

type quotaDb interface {
	GetQuotaUsedList(address types.Address) []types.QuotaInfo
	GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
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

func CalcSnapshotCurrentQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (uint64, error) {
	_, _, _, snapshotCurrentQuota, _, _, err := nodeConfig.calcQuotaFunc(db, beneficial, pledgeAmount, big.NewInt(0))
	if err == nil || err == util.ErrInvalidUnconfirmedQuota {
		return snapshotCurrentQuota, nil
	}
	return 0, err
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (types.Quota, error) {
	quotaTotal, pledgeQuota, _, snapshotCurrentQuota, quotaAvg, blocked, err := nodeConfig.calcQuotaFunc(db, beneficial, pledgeAmount, big.NewInt(0))
	return types.NewQuota(pledgeQuota, quotaTotal, quotaAvg, snapshotCurrentQuota, blocked), err
}

func CalcQuotaForBlock(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition uint64, err error) {
	quotaTotal, _, quotaAddition, _, _, _, err = nodeConfig.calcQuotaFunc(db, addr, pledgeAmount, difficulty)
	if err != nil {
		return 0, 0, err
	}
	if quotaTotal > quotaLimitForBlock {
		if quotaAddition > quotaLimitForBlock {
			return quotaLimitForBlock, quotaLimitForBlock, nil
		} else {
			return quotaLimitForBlock, quotaAddition, nil
		}
	} else {
		return quotaTotal, quotaAddition, nil
	}
}

func CalcCreateQuota(fee *big.Int) uint64 {
	return quotaForCreateContract
}

// Check whether current quota of a contract account is enough to receive a new block
func CheckQuota(db quotaDb, q types.Quota, addr types.Address) bool {
	if q.Blocked() {
		return false
	}
	if q.Current() >= q.Avg() {
		return true
	} else {
		return false
	}
}

func calcQuotaV3(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, err error) {
	powFlag := difficulty != nil && difficulty.Sign() > 0
	if powFlag {
		canPoW, err := CanPoW(db, addr)
		if err != nil {
			return 0, 0, 0, 0, 0, false, err
		}
		if !canPoW {
			return 0, 0, 0, 0, 0, false, util.ErrCalcPoWTwice
		}
	}
	return calcQuotaTotal(db, addr, pledgeAmount, difficulty)
}
func isBlocked(db quotaDb, addr types.Address) (bool, error) {
	if !types.IsContractAddr(addr) {
		return false, nil
	}

	prevBlock, err := db.GetLatestAccountBlock(addr)
	if err != nil {
		return true, err
	}
	if prevBlock == nil {
		return false, nil
	}
	if prevBlock.BlockType == ledger.BlockTypeReceiveError {
		confirmTime, err := db.GetConfirmedTimes(prevBlock.Hash)
		if err != nil {
			return true, err
		}
		if confirmTime < outOfQuotaBlockTime {
			return true, nil
		}
	}
	return false, nil
}
func calcQuotaTotal(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, err error) {
	blocked, err = isBlocked(db, addr)
	if err != nil {
		return 0, 0, 0, 0, 0, false, err
	}

	quotaPledge, err = calcPledgeQuota(db, pledgeAmount)
	if err != nil {
		return 0, 0, 0, 0, 0, false, err
	}
	quotaAddition, err = calcPoWQuota(db, difficulty)
	if err != nil {
		return 0, 0, 0, 0, 0, false, err
	}
	quotaList := db.GetQuotaUsedList(addr)
	quotaTotal = uint64(0)
	quotaUsedTotal := uint64(0)
	blockCountTotal := uint64(0)
	if len(quotaList) > 1 {
		for _, q := range quotaList[:len(quotaList)-1] {
			quotaTotal = quotaTotal + quotaPledge
			quotaUsedTotal = quotaUsedTotal + q.QuotaUsedTotal
			blockCountTotal = blockCountTotal + q.BlockCount
			if quotaTotal >= q.QuotaTotal {
				quotaTotal = quotaTotal - q.QuotaTotal
			} else {
				quotaTotal = 0
			}
		}
	}
	var unconfirmedQuota uint64
	if len(quotaList) > 0 {
		q := quotaList[len(quotaList)-1]
		quotaTotal = quotaTotal + quotaPledge
		snapshotCurrentQuota = quotaTotal
		quotaUsedTotal = quotaUsedTotal + q.QuotaUsedTotal
		blockCountTotal = blockCountTotal + q.BlockCount
		unconfirmedQuota = q.QuotaTotal
	} else {
		snapshotCurrentQuota = quotaTotal
		unconfirmedQuota = 0
	}
	if blockCountTotal > 0 {
		quotaAvg = quotaUsedTotal / blockCountTotal
	} else {
		quotaAvg = 0
	}
	if quotaTotal >= unconfirmedQuota {
		quotaTotal = quotaTotal - unconfirmedQuota
	} else {
		return 0, quotaPledge, 0, snapshotCurrentQuota, quotaAvg, blocked, util.ErrInvalidUnconfirmedQuota
	}
	if blocked {
		return 0, quotaPledge, 0, snapshotCurrentQuota, quotaAvg, blocked, nil
	}
	return quotaTotal + quotaAddition, quotaPledge, quotaAddition, snapshotCurrentQuota, quotaAvg, blocked, nil
}

func calcPledgeQuota(db quotaDb, pledgeAmount *big.Int) (uint64, error) {
	if pledgeAmount == nil || pledgeAmount.Sign() <= 0 {
		return 0, nil
	}
	return calcQuotaByIndex(getIndexInBigIntList(pledgeAmount, nodeConfig.pledgeAmountList, 0, len(nodeConfig.sectionList)-1)), nil
}

func calcPoWQuota(db quotaDb, difficulty *big.Int) (uint64, error) {
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

func getIndexByQuota(q uint64) (int, error) {
	index := int((q + quotaForSection - 1) / quotaForSection)
	if index >= len(nodeConfig.sectionList) || uint64(index)*quotaForSection < q {
		return 0, util.ErrBlockQuotaLimitReached
	}
	return index, nil
}

func getIndexInBigIntList(x *big.Int, list []*big.Int, left, right int) int {
	if left == right {
		return getExactIndex(x, list, left)
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

func getExactIndex(x *big.Int, list []*big.Int, index int) int {
	if index == 0 || list[index].Cmp(x) <= 0 {
		return index
	} else {
		return index - 1
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

func CalcPoWDifficulty(quotaRequired uint64, q types.Quota) (*big.Int, error) {
	if quotaRequired > quotaLimitForBlock {
		return nil, util.ErrBlockQuotaLimitReached
	}
	if q.Current() >= quotaRequired {
		return big.NewInt(0), nil
	}
	index, err := getIndexByQuota(quotaRequired)
	if err != nil {
		return nil, err
	}
	difficulty := nodeConfig.difficultyList[index]
	return difficulty, nil
}
