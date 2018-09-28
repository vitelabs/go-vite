package quota

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
)

var (
	ErrOutOfQuota      = errors.New("out of quota")
	errGasUintOverflow = errors.New("gas uint64 overflow")

	quotaByPledge = big.NewInt(1e9)
)

const (
	quotaForPoW              uint64 = 21000
	quotaLimit               uint64 = 3000000 // Maximum quota of an account referring to one snapshot block
	quotaLimitForTransaction uint64 = 800000  // Maximum quota of a transaction
	TxGas                    uint64 = 21000   // Per transaction not creating a contract.
	txContractCreationGas    uint64 = 53000   // Per transaction that creates a contract.
	txDataZeroGas            uint64 = 4       // Per byte of data attached to a transaction that equals zero.
	txDataNonZeroGas         uint64 = 68      // Per byte of data attached to a transaction that is not equal to zero.
)

type quotaDb interface {
	contracts.StorageDatabase
	GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock
	CurrentSnapshotBlock() *ledger.SnapshotBlock
	PrevAccountBlock() *ledger.AccountBlock
	GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock
}

func GetPledgeQuota(db quotaDb, beneficial types.Address) uint64 {
	// TODO cache
	// TODO benchmark
	quotaTotal, _ := CalcQuota(db, beneficial, false)
	return quotaTotal
}

// TODO pow bool to difficulty *big.Int
func CalcQuota(db quotaDb, addr types.Address, pow bool) (quotaTotal uint64, quotaAddition uint64) {
	// quotaInit = quotaLimitForAccount * (1 - 2/(1 + e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount)))
	// 				- quota used by prevBlock referring to the same snapshot hash
	// quotaAddition = quotaLimitForAccount * (1 - 2/(1 + e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount)))
	//				- quotaLimitForAccount * (1 - 2/(1 + e**(fPledge * snapshotHeightGap * pledgeAmount)))
	// e**(fDifficulty * difficulty + fPledge * snapshotHeightGap * pledgeAmount) is discrete to reduce computation complexity
	// quotaLimitForAccount is within a range decided by net congestion and net capacity
	// user account gets extra quota to send or receive a transaction if calc PoW, extra quota num is according to difficulty
	// contract account only gets quota via pledge
	// user account genesis block must calculate a PoW to get quota
	// TODO
	// The following code is just a simple implementation for test net.
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
				quotaInitBig.Mul(quotaInitBig, new(big.Int).SetUint64(db.CurrentSnapshotBlock().Height))
			} else {
				quotaInitBig.Mul(quotaInitBig, new(big.Int).SetUint64(db.CurrentSnapshotBlock().Height-db.GetSnapshotBlockByHash(&prevBlock.SnapshotHash).Height))
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
}

func CalcCreateQuota(fee *big.Int) uint64 {
	quota := new(big.Int).Div(fee, quotaByPledge)
	if quota.IsUint64() {
		return helper.Min(quotaLimitForTransaction, quota.Uint64())
	}
	return quotaLimitForTransaction
}

func IntrinsicGasCost(data []byte, isCreate bool) (uint64, error) {
	var gas uint64
	if isCreate {
		gas = txContractCreationGas
	} else {
		gas = TxGas
	}
	gasData, err := DataGasCost(data)
	if err != nil || helper.MaxUint64-gas < gasData {
		return 0, errGasUintOverflow
	}
	return gas + gasData, nil
}

func DataGasCost(data []byte) (uint64, error) {
	var gas uint64
	if len(data) > 0 {
		var nonZeroByteCount uint64
		for _, byteCode := range data {
			if byteCode != 0 {
				nonZeroByteCount++
			}
		}
		if helper.MaxUint64/txDataNonZeroGas < nonZeroByteCount {
			return 0, errGasUintOverflow
		}
		gas = nonZeroByteCount * txDataNonZeroGas

		zeroByteCount := uint64(len(data)) - nonZeroByteCount
		if (helper.MaxUint64-gas)/txDataZeroGas < zeroByteCount {
			return 0, errGasUintOverflow
		}
		gas += zeroByteCount * txDataZeroGas
	}
	return gas, nil
}

func QuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund uint64, err error) uint64 {
	if err == ErrOutOfQuota {
		return quotaTotal - quotaAddition
	} else if err != nil {
		if quotaTotal-quotaLeft < quotaAddition {
			return 0
		} else {
			return quotaTotal - quotaAddition - quotaLeft
		}
	} else {
		if quotaTotal-quotaLeft < quotaAddition {
			return 0
		} else {
			return quotaTotal - quotaLeft - quotaAddition - helper.Min(quotaRefund, (quotaTotal-quotaAddition-quotaLeft)/2)
		}
	}
}

func UseQuota(quotaLeft, cost uint64) (uint64, error) {
	if quotaLeft < cost {
		return 0, ErrOutOfQuota
	}
	quotaLeft = quotaLeft - cost
	return quotaLeft, nil
}
func UseQuotaForData(data []byte, quotaLeft uint64) (uint64, error) {
	cost, err := DataGasCost(data)
	if err != nil {
		return 0, err
	}
	return UseQuota(quotaLeft, cost)
}

func IsPoW(nonce []byte) bool {
	return len(nonce) > 0
}
