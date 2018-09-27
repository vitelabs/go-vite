package vm

import "errors"

var (
	ErrInsufficientBalance         = errors.New("insufficient balance for transfer")
	ErrContractAddressCreationFail = errors.New("contract address creation fail")
	ErrAddressCollision            = errors.New("contract address collision")
	ErrIdCollision                 = errors.New("id collision")
	ErrExecutionReverted           = errors.New("execution reverted")
	ErrInvalidData                 = errors.New("invalid data")
)

var (
	errGasUintOverflow       = errors.New("gas uint64 overflow")
	errReturnDataOutOfBounds = errors.New("evm: return data out of bounds")
)
