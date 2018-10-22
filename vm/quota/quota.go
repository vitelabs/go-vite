package quota

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

type quotaDb interface {
	GetStorage(addr *types.Address, key []byte) []byte
	NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator
	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock
	GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) uint64 {
	// TODO cache
	quotaTotal, _ := CalcQuota(db, beneficial, pledgeAmount, false)
	return quotaTotal
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
func CalcQuota(db quotaDb, addr types.Address, pledgeAmount *big.Int, pow bool) (quotaTotal uint64, quotaAddition uint64) {
	if pow {
		return CalcQuotaV2(db, addr, pledgeAmount, DefaultDifficulty)
	} else {
		return CalcQuotaV2(db, addr, pledgeAmount, helper.Big0)
	}
}

/*func CalcQuotaV1(db quotaDb, addr types.Address, pow bool) (quotaTotal uint64, quotaAddition uint64) {
	// Following code is just a simple implementation for test net.
	defer monitor.LogTime("vm", "CalcQuota", time.Now())
	quotaLimitForAccount := quotaLimit
	quotaInitBig := new(big.Int).Div(contracts.GetPledgeBeneficialAmount(db, addr), quotaByPledge)
	quotaAddition = uint64(0)
	if pow {
		quotaAddition = quotaForPoW
	}
	currentSnapshotHash := db.CurrentSnapshotBlock().Hash
	prevBlock := db.PrevAccountBlock()
	quotaUsed := uint64(0)
	for {
		if prevBlock != nil && currentSnapshotHash == prevBlock.SnapshotHash {
			// quick fail on a receive error block referencing to the same snapshot block
			// only one block gets extra quota when referencing to the same snapshot block
			if prevBlock.BlockType == ledger.BlockTypeReceiveError || (len(prevBlock.Nonce) > 0 && pow) {
				return 0, 0
			}
			quotaUsed = quotaUsed + prevBlock.Quota
			prevBlock = db.GetAccountBlockByHash(&prevBlock.PrevHash)
		} else {
			if prevBlock == nil {
				// first account block or first few account blocks referring to the same snapshot block
				quotaInitBig.Mul(quotaInitBig, helper.Big1)
			} else {
				quotaInitBig.Mul(quotaInitBig, new(big.Int).SetUint64(helper.Min(maxQuotaHeightGap, db.CurrentSnapshotBlock().Height-db.GetSnapshotBlockByHash(&prevBlock.SnapshotHash).Height)))
			}
			if quotaInitBig.BitLen() > 64 {
				quotaTotal = quotaLimitForAccount
			} else {
				quotaTotal = helper.Min(quotaInitBig.Uint64(), quotaLimitForAccount)
			}
			if quotaTotal < quotaUsed {
				return 0, 0
			}
			quotaTotal = quotaTotal - quotaUsed
			if quotaLimitForAccount-quotaAddition < quotaTotal {
				quotaAddition = quotaLimitForAccount - quotaTotal
				quotaTotal = quotaLimitForAccount
			} else {
				quotaTotal = quotaTotal + quotaAddition
			}
			return quotaTotal, quotaAddition
		}
	}
}*/

func CalcCreateQuota(fee *big.Int) uint64 {
	// TODO calc create quota
	return quotaForCreateContract
}

func IsPoW(nonce []byte) bool {
	return len(nonce) > 0
}

func calcQuotaInSections(pledgeAmount *big.Int, difficulty *big.Int) uint64 {
	x := big.NewFloat(0)
	floatTmp := new(big.Float)
	if pledgeAmount.Sign() > 0 {
		floatTmp.SetInt(pledgeAmount)
		x.Mul(paramA, floatTmp)
	}
	if difficulty.Sign() > 0 {
		floatTmp.SetInt(difficulty)
		floatTmp.Mul(paramB, floatTmp)
		x.Add(x, floatTmp)
	}
	return 0
}

var DefaultDifficulty = new(big.Int).SetUint64(0xffffffc000000000)

func CalcQuotaV2(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (uint64, uint64) {
	isPoW := difficulty.Sign() > 0
	currentSnapshotHash := db.CurrentSnapshotBlock().Hash
	prevBlock := db.PrevAccountBlock()
	quotaUsed := uint64(0)
	for {
		if prevBlock != nil && currentSnapshotHash == prevBlock.SnapshotHash {
			// quick fail on a receive error block referencing to the same snapshot block
			if prevBlock.BlockType == ledger.BlockTypeReceiveError {
				return 0, 0
			}
			if isPoW && IsPoW(prevBlock.Nonce) {
				// only one block gets extra quota when referencing to the same snapshot block
				return 0, 0
			}
			quotaUsed = quotaUsed + prevBlock.Quota
			prevBlock = db.GetAccountBlockByHash(&prevBlock.PrevHash)
		} else {
			x := new(big.Float).SetPrec(precForFloat).SetUint64(0)
			tmpFLoat := new(big.Float).SetPrec(precForFloat)
			var quotaWithoutPoW uint64
			if prevBlock == nil || pledgeAmount.Sign() == 0 {
				// first account block or first few account blocks referring to the same snapshot block
				quotaWithoutPoW = 0
			} else {
				tmpFLoat.SetUint64(helper.Min(maxQuotaHeightGap, db.CurrentSnapshotBlock().Height-db.GetSnapshotBlockByHash(&prevBlock.SnapshotHash).Height))
				x.Mul(tmpFLoat, paramA)
				tmpFLoat.SetInt(pledgeAmount)
				x.Mul(tmpFLoat, x)
				quotaWithoutPoW = uint64(getIndexInSection(x)) * quotaForSection
			}
			if quotaWithoutPoW < quotaUsed {
				return 0, 0
			}
			quotaTotal := quotaWithoutPoW
			if isPoW {
				tmpFLoat.SetInt(difficulty)
				tmpFLoat.Mul(tmpFLoat, paramB)
				x.Add(x, tmpFLoat)
				quotaTotal = uint64(getIndexInSection(x)) * quotaForSection
			}
			return quotaTotal - quotaUsed, quotaTotal - quotaWithoutPoW
		}
	}
}

// Get the largest index
// which makes sectionList[index] <= x
func getIndexInSection(x *big.Float) int {
	return getIndexInSectionRange(x, 0, len(sectionList)-1)
}
func getIndexInSectionRange(x *big.Float, left, right int) int {
	if left == right {
		return getExactIndex(x, left)
	}
	mid := (left + right + 1) / 2
	cmp := sectionList[mid].Cmp(x)
	if cmp == 0 {
		return mid
	} else if cmp > 0 {
		return getIndexInSectionRange(x, left, mid-1)
	} else {
		return getIndexInSectionRange(x, mid, right)
	}
}

func getExactIndex(x *big.Float, index int) int {
	if sectionList[index].Cmp(x) <= 0 || index == 0 {
		return index
	} else {
		return index - 1
	}
}
