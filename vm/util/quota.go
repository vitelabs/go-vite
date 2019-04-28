package util

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	txDataGas             uint64 = 68
	TxGas                 uint64 = 21000 // Per transaction not creating a contract.
	txContractCreationGas uint64 = 53000 // Per transaction that creates a contract.
	QuotaRange            uint64 = 75
	ConfirmGas            uint64 = 200
	CommonQuotaRatio      uint8  = 10
	QuotaRatioDivision    uint64 = 10
)

func MultipleCost(cost uint64, quotaRatio uint8) (uint64, error) {
	if quotaRatio < CommonQuotaRatio {
		return 0, ErrInvalidQuotaRatio
	}
	if quotaRatio == CommonQuotaRatio {
		return cost, nil
	}
	ratioUint64 := uint64(quotaRatio)
	if cost > helper.MaxUint64/ratioUint64 {
		return 0, ErrGasUintOverflow
	}
	return cost * ratioUint64 / QuotaRatioDivision, nil
}

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
	if l := uint64(len(data)); l > 0 {
		if helper.MaxUint64/txDataGas < l {
			return 0, ErrGasUintOverflow
		}
		gas = l * txDataGas
	}
	return gas, nil
}

func TotalGasCost(baseCost uint64, data []byte) (uint64, error) {
	dataCost, err := DataGasCost(data)
	if err != nil {
		return 0, err
	}
	totalCost, overflow := helper.SafeAdd(baseCost, dataCost)
	if overflow {
		return 0, err
	}
	return totalCost, nil
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
