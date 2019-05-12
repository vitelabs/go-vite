package util

type VMError struct {
	errMsg     string
	costAllGas bool
}

func (e VMError) Error() string {
	return e.errMsg
}
func (e VMError) CostAllGas() bool {
	return e.costAllGas
}

var (
	ErrInvalidMethodParam        = VMError{"invalid method param", false}
	ErrInvalidQuotaRatio         = VMError{"invalid quota ratio", false}
	ErrRewardIsNotDrained        = VMError{"reward is not drained", false}
	ErrInsufficientBalance       = VMError{"insufficient balance for transfer", false}
	ErrCalcPoWTwice              = VMError{"calc PoW twice referring to one snapshot block", false}
	ErrAbiMethodNotFound         = VMError{"abi: method not found", false}
	ErrInvalidConfirmTime        = VMError{"invalid confirm time", false}
	ErrAddressNotMatch           = VMError{"current address not match", false}
	ErrTransactionTypeNotSupport = VMError{"transaction type not supported", false}
	ErrVersionNotSupport         = VMError{"feature not supported in current snapshot height", false}
	ErrBlockTypeNotSupported     = VMError{"block type not supported", true}
	ErrDataNotExist              = VMError{"data not exist", false}
	ErrContractNotExists         = VMError{"contract not exists", false}
	ErrNoReliableStatus          = VMError{"no reliable status", false}

	ErrAddressCollision = VMError{"contract address collision", false}
	ErrIdCollision      = VMError{"id collision", false}
	ErrRewardNotDue     = VMError{"reward not due", false}

	ErrExecutionReverted = VMError{"execution reverted", false}
	ErrDepth             = VMError{"max call depth exceeded", false}

	ErrGasUintOverflow          = VMError{"gas uint64 overflow", true}
	ErrMemSizeOverflow          = VMError{"memory size uint64 overflow", true}
	ErrReturnDataOutOfBounds    = VMError{"vm: return data out of bounds", true}
	ErrBlockQuotaLimitReached   = VMError{"quota limit for block reached", true}
	ErrAccountQuotaLimitReached = VMError{"quota limit for account reached", true}
	ErrOutOfQuota               = VMError{"out of quota", true}
	ErrInvalidUnconfirmedQuota  = VMError{"calc quota failed, invalid unconfirmed quota", true}

	ErrStackLimitReached      = VMError{"stack limit reached", true}
	ErrStackUnderflow         = VMError{"stack underflow", true}
	ErrInvalidJumpDestination = VMError{"invalid jump destination", true}
	ErrInvalidOpCode          = VMError{"invalid opcode", true}

	ErrChainForked          = VMError{"chain forked", false}
	ErrContractCreationFail = VMError{"contract creation failed", false}
)

func DealWithErr(v interface{}) {
	if v != nil {
		panic(v)
	}
}
