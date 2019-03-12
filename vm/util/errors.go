package util

import "errors"

var (
	ErrInvalidMethodParam          = errors.New("invalid method param")
	ErrInsufficientBalance         = errors.New("insufficient balance for transfer")
	ErrContractAddressCreationFail = errors.New("contract address creation fail")
	ErrAddressCollision            = errors.New("contract address collision")
	ErrIdCollision                 = errors.New("id collision")
	ErrExecutionReverted           = errors.New("execution reverted")
	ErrGasUintOverflow             = errors.New("gas uint64 overflow")
	ErrMemSizeOverflow             = errors.New("memory size uint64 overflow")
	ErrReturnDataOutOfBounds       = errors.New("vm: return data out of bounds")
	ErrCalcPoWTwice                = errors.New("calc PoW twice referring to one snapshot block")
	ErrAbiMethodNotFound           = errors.New("abi: method not found")
	ErrDepth                       = errors.New("max call depth exceeded")
	ErrInvalidConfirmTime          = errors.New("invalid confirm time")

	ErrForked                     = errors.New("chain forked")
	ErrCalcPoWLimitReached        = errors.New("can not calc PoW in this block")
	ErrContractSendBlockRunFailed = errors.New("contract send block run failed")
	ErrVersionNotSupport          = errors.New("feature not supported in current snapshot height")
)
