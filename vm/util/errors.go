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
	ErrReturnDataOutOfBounds       = errors.New("evm: return data out of bounds")
	ErrCalcPoWTwice                = errors.New("calc PoW twice referring to one snapshot block")
)
