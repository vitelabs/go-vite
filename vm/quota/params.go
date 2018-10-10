package quota

import (
	"errors"
	"math/big"
)

var (
	ErrOutOfQuota      = errors.New("out of quota")
	errGasUintOverflow = errors.New("gas uint64 overflow")

	quotaByPledge = big.NewInt(1e18)
)

const (
	quotaForPoW            uint64 = 21000
	quotaLimit             uint64 = 3000000   // Maximum quota of an account referring to one snapshot block.
	quotaForCreateContract uint64 = 800000    // Quota limit for create contract.
	TxGas                  uint64 = 21000     // Per transaction not creating a contract.
	txContractCreationGas  uint64 = 53000     // Per transaction that creates a contract.
	txDataZeroGas          uint64 = 4         // Per byte of data attached to a transaction that equals zero.
	txDataNonZeroGas       uint64 = 68        // Per byte of data attached to a transaction that is not equal to zero.
	maxQuotaHeightGap      uint64 = 3600 * 24 // Maximum Snapshot block height gap to gain quota by pledge.
)
