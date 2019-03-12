package quota

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

type NodeConfig struct {
	QuotaParams
	sectionList      []*big.Float
	difficultyList   []*big.Int
	pledgeAmountList []*big.Int
}

var nodeConfig NodeConfig

func InitQuotaConfig(isTestParam bool) {
	sectionList := make([]*big.Float, len(sectionStrList))
	for i, str := range sectionStrList {
		sectionList[i], _ = new(big.Float).SetPrec(precForFloat).SetString(str)
	}
	if isTestParam {
		nodeConfig = NodeConfig{QuotaParamTestnet, sectionList, difficultyListTestnet, pledgeAmountListTestnet}
	} else {
		nodeConfig = NodeConfig{QuotaParamMainnet, sectionList, difficultyListMainnet, pledgeAmountListMainnet}
	}
}

type quotaDb interface {
	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock
	//TODO GetCurrentQuota(addr types.Address) (uint64, int)
}

func GetPledgeQuota(db quotaDb, beneficial types.Address, pledgeAmount *big.Int) (types.Quota, error) {
	quotaTotal, _, quotaUsed, quotaAvg, err := calcQuotaV3(db, beneficial, pledgeAmount, big.NewInt(0))
	return types.NewQuota(quotaTotal, quotaUsed, quotaAvg), err
}

func CalcQuotaForBlock(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition uint64, err error) {
	quotaTotal, quotaAddition, quotaUsed, _, err := calcQuotaV3(db, addr, pledgeAmount, difficulty)
	if err != nil {
		return 0, 0, err
	}
	if quotaTotal <= quotaUsed {
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
	return current, quotaAddition, err
}

func CalcCreateQuota(fee *big.Int) uint64 {
	return quotaForCreateContract
}

// Check whether current quota of a contract account is enough to receive a new block
func CheckQuota(db quotaDb, addr types.Address, q types.Quota) bool {
	// TODO optimize for receive error
	if q.Current() >= q.Avg() {
		return true
	} else {
		return false
	}
}

func calcQuotaV3(db quotaDb, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal, quotaAddition, quotaUsed, quotaAvg uint64, err error) {
	powFlag := difficulty != nil && difficulty.Sign() > 0
	if powFlag && !CanPoW(db, addr) {
		return 0, 0, 0, 0, util.ErrCalcPoWTwice
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
		return (calcQuotaByPledgeAmountAndDifficulty(pledgeAmount, difficulty))
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
	// TODO
	return 0, 0
	/*quotaUsed, txNum := db.GetCurrentQuota(addr)
	if txNum == 0 {
		return 0, 0
	} else {
		return quotaUsed, quotaUsed / uint64(txNum)
	}*/
}

func calcQuotaByIndex(index int) uint64 {
	if index >= 0 && index < len(nodeConfig.sectionList) {
		return uint64(index) * quotaForSection
	}
	return 0
}

// Get the largest index
// which makes sectionList[index] <= x
func getIndexInSection(x *big.Float) int {
	return getIndexInFloatList(x, nodeConfig.sectionList, 0, len(nodeConfig.sectionList)-1)
}
func getIndexInFloatList(x *big.Float, list []*big.Float, left, right int) int {
	// TODO optimize according to quota formula
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
	// TODO optimize according to quota formula
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

func IsPoW(nonce []byte) bool {
	return len(nonce) > 0
}

func CanPoW(db quotaDb, addr types.Address) bool {
	// TODO use chain.getUnconfirmedAccountBlocks instead
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

func GetQuotaRequired(block *ledger.AccountBlock) (uint64, error) {
	if block.BlockType == ledger.BlockTypeSendCreate {
		quotaRequired, _ := util.IntrinsicGasCost(block.Data, false)
		return quotaRequired, nil
	} else if block.BlockType == ledger.BlockTypeReceive {
		quotaRequired, _ := util.IntrinsicGasCost(nil, false)
		return quotaRequired, nil
	} else if block.BlockType == ledger.BlockTypeSendCall {
		if types.IsBuiltinContractAddrInUse(block.ToAddress) {
			if method, ok, err := contracts.GetBuiltinContract(block.ToAddress, block.Data); !ok || err != nil {
				return 0, errors.New("built-in contract method not exists")
			} else {
				return method.GetSendQuota(block.Data)
			}
		} else {
			quotaRequired, _ := util.IntrinsicGasCost(block.Data, false)
			return quotaRequired, nil
		}
	} else {
		return 0, errors.New("block type not supported")
	}
}

func CalcPoWDifficulty(quotaRequired uint64, q types.Quota, pledgeAmount *big.Int) (*big.Int, error) {
	expectedTotal := quotaRequired + q.Used()
	if expectedTotal > quotaLimitForAccount {
		return nil, errors.New("quota limit reached")
	}
	tmpFloat := new(big.Float).SetPrec(precForFloat).SetInt(pledgeAmount)
	tmpFloat = tmpFloat.Mul(tmpFloat, nodeConfig.paramA)
	tmpFloat = tmpFloat.Sub(nodeConfig.sectionList[0], tmpFloat)
	tmpFloat = tmpFloat.Quo(tmpFloat, nodeConfig.paramB)
	difficulty, _ := tmpFloat.Int(nil)
	difficulty = difficulty.Add(difficulty, helper.Big1)
	for calcQuotaByPledgeAmountAndDifficulty(pledgeAmount, difficulty) < expectedTotal {
		difficulty.Add(difficulty, helper.Big1)
	}
	return difficulty, nil
}
