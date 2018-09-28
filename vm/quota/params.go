package quota

import (
	"errors"
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
