package quota

import (
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

type quotaConfigParams struct {
	difficultyList  []*big.Int
	stakeAmountList []*big.Int
	qcIndexMin      uint64
	qcIndexMax      uint64
	qcMap           map[uint64]*big.Int
	calcQuotaFunc   func(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error)
}

var quotaConfig quotaConfigParams

// InitQuotaConfig init global status of quota calculation. This method is
// supposed be called when the node started.
func InitQuotaConfig(isTest, isTestParam bool) {
	if isTestParam {
		quotaConfig = quotaConfigParams{
			difficultyList: difficultyListTestnet}
		stakeAmountList := make([]*big.Int, len(stakeAmountListTestnet))
		for i, str := range stakeAmountListTestnet {
			stakeAmountList[i], _ = new(big.Int).SetString(str, 10)
		}
		quotaConfig.stakeAmountList = stakeAmountList
	} else {
		quotaConfig = quotaConfigParams{
			difficultyList: difficultyListMainnet}
		stakeAmountList := make([]*big.Int, len(stakeAmountListMainnet))
		for i, str := range stakeAmountListMainnet {
			stakeAmountList[i], _ = new(big.Int).SetString(str, 10)
		}
		quotaConfig.stakeAmountList = stakeAmountList
	}

	quotaConfig.qcIndexMin = qcIndexMinMainnet
	quotaConfig.qcIndexMax = qcIndexMaxMainnet
	quotaConfig.qcMap = qcMapMainnet

	if isTest {
		quotaConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error) {
			return 75000000, 1000000, 0, 1000000, 0, false, 0, nil
		}
	} else {
		quotaConfig.calcQuotaFunc = func(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error) {
			return calcQuotaV3(db, addr, stakeAmount, difficulty, sbHeight)
		}
	}
}

type quotaDb interface {
	GetGlobalQuota() types.QuotaInfo
	GetQuotaUsedList(address types.Address) []types.QuotaInfo
	GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
}

// CalcBlockQuotaUsed recalculate quotaUsed field of an account block
func CalcBlockQuotaUsed(db quotaDb, block *ledger.AccountBlock, sbHeight uint64) (uint64, error) {
	if !util.IsPoW(block) {
		return block.QuotaUsed, nil
	}
	powQuota := calcPoWQuotaByQc(db, block.Difficulty, sbHeight)
	if block.QuotaUsed > powQuota {
		return block.QuotaUsed - powQuota, nil
	}
	return 0, nil
}

// GetSnapshotCurrentQuota calculate available quota for an account, excluding unconfirmed blocks
func GetSnapshotCurrentQuota(db quotaDb, beneficial types.Address, stakeAmount *big.Int, sbHeight uint64) (uint64, error) {
	_, _, _, snapshotCurrentQuota, _, _, _, err := quotaConfig.calcQuotaFunc(db, beneficial, stakeAmount, big.NewInt(0), sbHeight)
	if err == nil || err == util.ErrInvalidUnconfirmedQuota {
		return snapshotCurrentQuota, nil
	}
	return 0, err
}

// GetQuota calculate available quota for an account
func GetQuota(db quotaDb, beneficial types.Address, stakeAmount *big.Int, sbHeight uint64) (types.Quota, error) {
	quotaTotal, stakeQuota, _, snapshotCurrentQuota, quotaAvg, blocked, blockReleaseHeight, err := quotaConfig.calcQuotaFunc(db, beneficial, stakeAmount, big.NewInt(0), sbHeight)
	return types.NewQuota(stakeQuota, quotaTotal, quotaAvg, snapshotCurrentQuota, blocked, blockReleaseHeight), err
}

// GetQuotaForBlock calculate available quota for a block
func GetQuotaForBlock(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaAddition uint64, err error) {
	quotaTotal, _, quotaAddition, _, _, _, _, err = quotaConfig.calcQuotaFunc(db, addr, stakeAmount, difficulty, sbHeight)
	if err != nil {
		return 0, 0, err
	}
	if quotaTotal > quotaLimitForBlock {
		if quotaAddition > quotaLimitForBlock {
			return quotaLimitForBlock, quotaLimitForBlock, nil
		}
		return quotaLimitForBlock, quotaAddition, nil
	}
	return quotaTotal, quotaAddition, nil
}

// CheckQuota check whether current quota of a contract account is enough to receive a new block
func CheckQuota(db quotaDb, q types.Quota, addr types.Address) (bool, uint64) {
	if q.Blocked() {
		return false, q.BlockReleaseHeight()
	}
	if q.Current() >= q.Avg() {
		return true, 0
	} else {
		return false, 1
	}
}

func calcQuotaV3(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error) {
	powFlag := difficulty != nil && difficulty.Sign() > 0
	if powFlag {
		canPoW := CanPoW(db, addr)
		if !canPoW {
			return 0, 0, 0, 0, 0, false, 0, util.ErrCalcPoWTwice
		}
	}
	return calcQuotaTotal(db, addr, stakeAmount, difficulty, sbHeight)
}
func isBlocked(db quotaDb, addr types.Address) (bool, uint64, error) {
	if !types.IsContractAddr(addr) {
		return false, 0, nil
	}

	prevBlock, err := db.GetLatestAccountBlock(addr)
	if err != nil {
		return true, 0, err
	}
	if prevBlock == nil {
		return false, 0, nil
	}
	if prevBlock.BlockType == ledger.BlockTypeReceiveError {
		confirmTime, err := db.GetConfirmedTimes(prevBlock.Hash)
		if err != nil {
			return true, 0, err
		}
		if confirmTime < outOfQuotaBlockTime {
			return true, outOfQuotaBlockTime - confirmTime, nil
		}
	}
	return false, 0, nil
}
func calcQuotaTotal(db quotaDb, addr types.Address, stakeAmount *big.Int, difficulty *big.Int, sbHeight uint64) (quotaTotal, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg uint64, blocked bool, blockReleaseHeight uint64, err error) {
	blocked, blockReleaseHeight, err = isBlocked(db, addr)
	if err != nil {
		return 0, 0, 0, 0, 0, false, 0, err
	}
	qc, _, isCongestion := CalcQc(db, sbHeight)
	quotaStake = calcStakeQuota(qc, isCongestion, stakeAmount)
	quotaAddition = calcPoWQuota(qc, isCongestion, difficulty)
	quotaList := db.GetQuotaUsedList(addr)
	quotaTotal = uint64(0)
	quotaUsedTotal := uint64(0)
	blockCountTotal := uint64(0)
	if len(quotaList) > 1 {
		for _, q := range quotaList[:len(quotaList)-1] {
			quotaTotal = quotaTotal + quotaStake
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
		quotaTotal = quotaTotal + quotaStake
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
		return 0, quotaStake, 0, snapshotCurrentQuota, quotaAvg, blocked, blockReleaseHeight, util.ErrInvalidUnconfirmedQuota
	}
	if blocked {
		return 0, quotaStake, 0, snapshotCurrentQuota, quotaAvg, blocked, blockReleaseHeight, nil
	}
	return quotaTotal + quotaAddition, quotaStake, quotaAddition, snapshotCurrentQuota, quotaAvg, blocked, blockReleaseHeight, nil
}

func calcStakeQuota(qc *big.Int, isCongestion bool, stakeAmount *big.Int) uint64 {
	if stakeAmount == nil || stakeAmount.Sign() <= 0 {
		return 0
	}
	stakeAmount = calcStakeParam(qc, isCongestion, stakeAmount)
	return calcQuotaByIndex(getIndexInBigIntList(stakeAmount, quotaConfig.stakeAmountList, 0, sectionLen))
}

func calcPoWQuotaByQc(db quotaDb, difficulty *big.Int, sbHeight uint64) uint64 {
	if difficulty == nil || difficulty.Sign() <= 0 {
		return 0
	}
	difficulty = calcStakeParamByQc(db, difficulty, sbHeight)
	return calcQuotaByIndex(getIndexInBigIntList(difficulty, quotaConfig.difficultyList, 0, sectionLen))
}

func calcPoWQuota(qc *big.Int, isCongestion bool, difficulty *big.Int) uint64 {
	if difficulty == nil || difficulty.Sign() <= 0 {
		return 0
	}
	difficulty = calcStakeParam(qc, isCongestion, difficulty)
	return calcQuotaByIndex(getIndexInBigIntList(difficulty, quotaConfig.difficultyList, 0, sectionLen))
}

func calcQuotaByIndex(index int) uint64 {
	if index >= 0 && index <= sectionLen {
		return uint64(index) * quotaForSection
	}
	return 0
}

func getIndexByQuota(q uint64) (int, error) {
	index := int((q + quotaForSection - 1) / quotaForSection)
	if index > sectionLen || uint64(index)*quotaForSection < q {
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
	}
	return index - 1
}

// CanPoW checks whether an account address can calculate pow in current snapshot block
func CanPoW(db quotaDb, address types.Address) bool {
	blocks := db.GetUnconfirmedBlocks(address)
	for _, b := range blocks {
		if util.IsPoW(b) {
			return false
		}
	}
	return true
}

// CalcPoWDifficulty calculate pow difficulty by quota
func CalcPoWDifficulty(db quotaDb, quotaRequired uint64, q types.Quota, sbHeight uint64) (*big.Int, error) {
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
	difficulty := new(big.Int).Set(quotaConfig.difficultyList[index])
	difficultyByQc, _, err := calcStakeTargetParamByQc(db, difficulty, sbHeight)
	return difficultyByQc, err
}

// CalcStakeAmountByQuota calculate stake amount by expected quota used per second
func CalcStakeAmountByQuota(q uint64) (*big.Int, error) {
	if q > getMaxQutoa() {
		return nil, util.ErrInvalidMethodParam
	} else if q == 0 {
		return big.NewInt(0), nil
	}
	index := (q + quotaForSection - 1) / quotaForSection
	return new(big.Int).Set(quotaConfig.stakeAmountList[index]), nil
}

func calcStakeTargetParamByQc(db quotaDb, target *big.Int, sbHeight uint64) (*big.Int, *big.Int, error) {
	qc, _, isCongestion := CalcQc(db, sbHeight)
	if !isCongestion {
		return target, qc, nil
	}
	newTarget, err := calcStakeTargetParam(qc, isCongestion, target)
	return newTarget, qc, err
}
func calcStakeTargetParam(qc *big.Int, isCongestion bool, target *big.Int) (*big.Int, error) {
	newTarget := new(big.Int).Mul(target, qcDivision)
	newTarget.Div(newTarget, qc)
	// following for loop will only execute once
	for true {
		calcTarget := calcStakeParam(qc, isCongestion, newTarget)
		if calcTarget.Cmp(target) >= 0 {
			break
		}
		newTarget.Add(newTarget, helper.Big1)
	}
	if newTarget.BitLen() <= 256 {
		return newTarget, nil
	}
	return nil, util.ErrBlockQuotaLimitReached
}

func getMaxQutoa() uint64 {
	return uint64(sectionLen) * quotaForSection
}

func calcStakeParamByQc(db quotaDb, param *big.Int, sbHeight uint64) *big.Int {
	qc, _, isCongestion := CalcQc(db, sbHeight)
	return calcStakeParam(qc, isCongestion, param)
}

func calcStakeParam(qc *big.Int, isCongestion bool, param *big.Int) *big.Int {
	if !isCongestion {
		return param
	}
	newParam := new(big.Int).Mul(param, qc)
	newParam = newParam.Div(newParam, qcDivision)
	return newParam
}

// CalcQc calculate quota congestion ratio
func CalcQc(db quotaDb, sbHeight uint64) (*big.Int, uint64, bool) {
	if !fork.IsDexFork(sbHeight) {
		return big.NewInt(0), 0, false
	}
	globalQuota := db.GetGlobalQuota().QuotaUsedTotal
	qcIndex := (globalQuota + qcGap - 1) / qcGap
	if qcIndex < quotaConfig.qcIndexMin {
		return qcDivision, globalQuota, false
	} else if qcIndex >= quotaConfig.qcIndexMax {
		qcIndex = quotaConfig.qcIndexMax
	}
	return quotaConfig.qcMap[qcIndex], globalQuota, true
}
