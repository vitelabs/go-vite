package util

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
)

var (
	ErrOutOfQuota      = errors.New("out of quota")
	errGasUintOverflow = errors.New("gas uint64 overflow")
)

const (
	txDataZeroGas               uint64 = 4     // Per byte of data attached to a transaction that equals zero.
	txDataNonZeroGas            uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero.
	TxGas                       uint64 = 21000 // Per transaction not creating a contract.
	PrecompiledContractsSendGas uint64 = 100000
	RefundGas                   uint64 = 21000
	txContractCreationGas       uint64 = 53000 // Per transaction that creates a contract.
)

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

func CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund uint64, err error) uint64 {
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
