package quota

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"time"
)

type NodeConfig struct {
	QuotaParams
	sectionList    []*big.Float
	difficultyList []*big.Int
	calcQuotaFunc  func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error)
}

var nodeConfig NodeConfig

func InitQuotaConfig(isTest bool, isTestParam bool) {
	sectionList := make([]*big.Float, len(sectionStrList))
	for i, str := range sectionStrList {
		sectionList[i], _ = new(big.Float).SetPrec(precForFloat).SetString(str)
	}
	var calcQuotaFunc func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error)
	if isTest {
		calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
			return 1000000, 0, nil
		}
	} else {
		calcQuotaFunc = func(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
			return calcQuota(db, addr, pledgeAmount, difficulty)
		}
	}
	if isTestParam {
		nodeConfig = NodeConfig{QuotaParamTest, sectionList, difficultyListTest, calcQuotaFunc}
	} else {
		nodeConfig = NodeConfig{QuotaParamMainNet, sectionList, difficultyListMainNet, calcQuotaFunc}
	}
}

type quotaDb interface {
	GetStorage(addr *types.Address, key []byte) []byte
	NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator
	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock
	GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (uint64, error) {
	// TODO cache
	quotaTotal, _, err := CalcQuota(db, beneficial, pledgeAmount, big.NewInt(0))
	return quotaTotal, err
}

// quotaInit = quotaLimitForAccount * (1 - 2/(1 + e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount)))
// 				- quota used by prevBlock referring to the same snapshot hash
// quotaAddition = quotaLimitForAccount * (1 - 2/(1 + e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount)))
//				- quotaLimitForAccount * (1 - 2/(1 + e**(fPledge * snapshotHeightGap * pledgeAmount)))
// snapshotHeightGap is limit to 1 day
// e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount) is discrete to reduce computation complexity
// quotaLimitForAccount is within a range decided by net congestion and net capacity
// user account gets extra quota to send or receive a transaction if calc PoW, extra quota is decided by difficulty
// contract account only gets quota via pledge
// user account genesis block(a receive block) must calculate a PoW to get quota
func CalcQuota(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
	return nodeConfig.calcQuotaFunc(db, addr, pledgeAmount, difficulty)
}
func calcQuota(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
	if difficulty != nil && difficulty.Sign() > 0 {
		/*if fork.IsLimitFork(db.CurrentSnapshotBlock().Height) {
			if powLimitReached, err := CheckPoWLimit(db); err != nil || powLimitReached {
				return 0, 0, util.ErrCalcPoWLimitReached
			}
		}*/
		return CalcQuotaV2(db, addr, pledgeAmount, difficulty)
	} else {
		return CalcQuotaV2(db, addr, pledgeAmount, helper.Big0)
	}
}

func CalcCreateQuota(fee *big.Int) uint64 {
	// TODO calc create quota
	return quotaForCreateContract
}

func IsPoW(nonce []byte) bool {
	return len(nonce) > 0
}

func CalcQuotaV2(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (uint64, uint64, error) {
	isPoW := difficulty.Sign() > 0
	currentSnapshotHash := db.CurrentSnapshotBlock().Hash
	prevBlock := db.PrevAccountBlock()
	quotaUsed := uint64(0)
	for {
		if prevBlock != nil && currentSnapshotHash == prevBlock.SnapshotHash {
			// quick fail on a receive error block referencing to the same snapshot block
			if prevBlock.BlockType == ledger.BlockTypeReceiveError {
				return 0, 0, util.ErrOutOfQuota
			}
			if isPoW && IsPoW(prevBlock.Nonce) {
				// only one block gets extra quota when referencing to the same snapshot block
				return 0, 0, util.ErrCalcPoWTwice
			}
			quotaUsed = quotaUsed + prevBlock.Quota
			prevBlock = db.GetAccountBlockByHash(&prevBlock.PrevHash)
		} else {
			x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
			tmpFLoat := new(big.Float).SetPrec(precForFloat)
			var quotaWithoutPoW uint64
			if pledgeAmount.Sign() == 0 {
				quotaWithoutPoW = 0
			} else {
				if prevBlock == nil {
					tmpFLoat.SetUint64(helper.Min(maxQuotaHeightGap, db.CurrentSnapshotBlock().Height))
				} else {
					prevSnapshotBlock := db.GetSnapshotBlockByHash(&prevBlock.SnapshotHash)
					if prevSnapshotBlock == nil {
						return 0, 0, util.ErrForked
					}
					tmpFLoat.SetUint64(helper.Min(maxQuotaHeightGap, db.CurrentSnapshotBlock().Height-prevSnapshotBlock.Height))
				}
				x.Mul(tmpFLoat, nodeConfig.paramA)
				tmpFLoat.SetInt(pledgeAmount)
				x.Mul(tmpFLoat, x)
				quotaWithoutPoW = calcQuotaInSection(x)
			}
			if quotaWithoutPoW < quotaUsed {
				return 0, 0, nil
			}
			quotaTotal := quotaWithoutPoW
			if isPoW {
				tmpFLoat.SetInt(difficulty)
				tmpFLoat.Mul(tmpFLoat, nodeConfig.paramB)
				x.Add(x, tmpFLoat)
				quotaTotal = calcQuotaInSection(x)
			}
			return quotaTotal - quotaUsed, quotaTotal - quotaWithoutPoW, nil
		}
	}
}

func calcQuotaInSection(x *big.Float) uint64 {
	// TODO calc Qm according to net congestion in past 3600 snapshot blocks
	return uint64(getIndexInSection(x)) * quotaForSection
}

// Get the largest index
// which makes sectionList[index] <= x
func getIndexInSection(x *big.Float) int {
	return getIndexInSectionRange(x, 0, len(nodeConfig.sectionList)-1)
}
func getIndexInSectionRange(x *big.Float, left, right int) int {
	if left == right {
		return getExactIndex(x, left)
	}
	mid := (left + right + 1) / 2
	cmp := nodeConfig.sectionList[mid].Cmp(x)
	if cmp == 0 {
		return mid
	} else if cmp > 0 {
		return getIndexInSectionRange(x, left, mid-1)
	} else {
		return getIndexInSectionRange(x, mid, right)
	}
}

func getExactIndex(x *big.Float, index int) int {
	if nodeConfig.sectionList[index].Cmp(x) <= 0 || index == 0 {
		return index
	} else {
		return index - 1
	}
}

// A single account is limited to send 10 tx with PoW in one day
func CheckPoWLimit(db quotaDb) (bool, error) {
	if db.PrevAccountBlock() == nil {
		return false, nil
	}
	powtimes := 1
	currentSb := db.CurrentSnapshotBlock()
	startTime := getTodayStartTime(currentSb.Timestamp, *db.GetGenesisSnapshotBlock().Timestamp)
	prevBlockHash := db.PrevAccountBlock().Hash
	for {
		prevBlock := db.GetAccountBlockByHash(&prevBlockHash)
		if prevBlock == nil || prevBlock.Timestamp.Before(*startTime) {
			return false, nil
		}
		if IsPoW(prevBlock.Nonce) {
			powtimes = powtimes + 1
			if powtimes > powTimesPerDay {
				return true, nil
			}
		}
		prevBlockHash = prevBlock.PrevHash
	}
}

func getTodayStartTime(currentTime *time.Time, genesisTime time.Time) *time.Time {
	startTime := genesisTime.Add(currentTime.Sub(genesisTime).Round(day))
	return &startTime
}
func calcSectionIndexByQuotaRequired(quotaRequired uint64) uint64 {
	if quotaRequired > helper.MaxUint64-quotaForSection+1 {
		return helper.MaxUint64/quotaForSection + 1
	}
	return (quotaRequired + quotaForSection - 1) / quotaForSection
}

func CanPoW(db quotaDb, addr types.Address) bool {
	currentSnapshotHash := db.CurrentSnapshotBlock().Hash
	prevBlock := db.PrevAccountBlock()
	for {
		if prevBlock != nil && currentSnapshotHash == prevBlock.SnapshotHash {
			if IsPoW(prevBlock.Nonce) {
				return false
			} else {
				prevBlock = db.GetAccountBlockByHash(&prevBlock.PrevHash)
				continue
			}
		} else {
			return true
		}
	}
}

func CalcPoWDifficulty(quotaRequired uint64) *big.Int {
	index := calcSectionIndexByQuotaRequired(quotaRequired)
	return new(big.Int).Set(nodeConfig.difficultyList[index])
}
