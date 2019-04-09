package util

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	txDataZeroGas         uint64 = 4     // Per byte of data attached to a transaction that equals zero.
	txDataNonZeroGas      uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero.
	TxGas                 uint64 = 21000 // Per transaction not creating a contract.
	txContractCreationGas uint64 = 53000 // Per transaction that creates a contract.
	QuotaRange            uint64 = 75
	ConfirmGas            uint64 = 200
)

func UseQuota(quotaLeft, cost uint64) (uint64, error) {
	if quotaLeft < cost {
		return 0, ErrOutOfQuota
	}
	quotaLeft = quotaLeft - cost
	return quotaLeft, nil
}

func UseQuotaWithFlag(quotaLeft, cost uint64, flag bool) (uint64, error) {
	if flag {
		return UseQuota(quotaLeft, cost)
	}
	return quotaLeft + cost, nil
}

func IntrinsicGasCost(data []byte, isCreate bool, confirmTime uint8) (uint64, error) {
	var gas uint64
	if isCreate {
		gas = txContractCreationGas
	} else {
		gas = TxGas
	}
	gasData, err := DataGasCost(data)
	if err != nil || helper.MaxUint64-gas < gasData {
		return 0, ErrGasUintOverflow
	}
	gas = gas + gasData
	if confirmTime == 0 {
		return gas, nil
	}
	confirmGas := uint64(confirmTime) * ConfirmGas
	if helper.MaxUint64-gas < confirmGas {
		return 0, ErrGasUintOverflow
	}
	return gas + confirmGas, nil
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
			return 0, ErrGasUintOverflow
		}
		gas = nonZeroByteCount * txDataNonZeroGas

		zeroByteCount := uint64(len(data)) - nonZeroByteCount
		if (helper.MaxUint64-gas)/txDataZeroGas < zeroByteCount {
			return 0, ErrGasUintOverflow
		}
		gas += zeroByteCount * txDataZeroGas
	}
	return gas, nil
}

func CalcQuotaUsed(useQuota bool, quotaTotal, quotaAddition, quotaLeft uint64, err error) uint64 {
	if !useQuota {
		return 0
	}
	if vmErr, ok := err.(VMError); ok && vmErr.CostAllGas() {
		return quotaTotal - quotaAddition
	} else {
		if quotaTotal-quotaLeft < quotaAddition {
			return 0
		} else {
			return quotaTotal - quotaAddition - quotaLeft
		}
	}
}

func IsPoW(block *ledger.AccountBlock) bool {
	return len(block.Nonce) > 0
}
