package util

import "errors"

var (
	ErrInvalidMethodParam        = errors.New("invalid method param")
	ErrInvalidQuotaMultiplier    = errors.New("invalid quota multiplier")
	ErrRewardIsNotDrained        = errors.New("reward is not drained")
	ErrInsufficientBalance       = errors.New("insufficient balance for transfer")
	ErrCalcPoWTwice              = errors.New("calc PoW twice referring to one snapshot block")
	ErrAbiMethodNotFound         = errors.New("abi: method not found")
	ErrInvalidResponseLatency    = errors.New("invalid response latency")
	ErrInvalidRandomDegree       = errors.New("invalid random degree")
	ErrAddressNotMatch           = errors.New("current address not match")
	ErrTransactionTypeNotSupport = errors.New("transaction type not supported")
	ErrVersionNotSupport         = errors.New("feature not supported in current snapshot height")
	ErrBlockTypeNotSupported     = errors.New("block type not supported")
	ErrDataNotExist              = errors.New("data not exist")
	ErrContractNotExists         = errors.New("contract not exists")
	ErrNoReliableStatus          = errors.New("no reliable status")

	ErrAddressCollision = errors.New("contract address collision")
	ErrIDCollision      = errors.New("id collision")
	ErrRewardNotDue     = errors.New("reward not due")

	ErrExecutionReverted = errors.New("execution reverted")
	ErrDepth             = errors.New("max call depth exceeded")

	ErrGasUintOverflow           = errors.New("gas uint64 overflow")
	ErrStorageModifyLimitReached = errors.New("contract storage modify count limit reached")
	ErrMemSizeOverflow           = errors.New("memory size uint64 overflow")
	ErrReturnDataOutOfBounds     = errors.New("vm: return data out of bounds")
	ErrBlockQuotaLimitReached    = errors.New("quota limit for block reached")
	ErrAccountQuotaLimitReached  = errors.New("quota limit for account reached")
	ErrOutOfQuota                = errors.New("out of quota")
	ErrInvalidCodeLength         = errors.New("invalid code length")
	ErrInvalidUnconfirmedQuota   = errors.New("calc quota failed, invalid unconfirmed quota")

	ErrStackLimitReached      = errors.New("stack limit reached")
	ErrStackUnderflow         = errors.New("stack underflow")
	ErrInvalidJumpDestination = errors.New("invalid jump destination")
	ErrInvalidOpCode          = errors.New("invalid opcode")

	ErrChainForked          = errors.New("chain forked")
	ErrContractCreationFail = errors.New("contract creation failed")

	ErrExecutionCanceled = errors.New("vm execution canceled")
)

// DealWithErr panics if err is not nil.
// Used when chain forked or db error.
func DealWithErr(v interface{}) {
	if v != nil {
		panic(v)
	}
}
