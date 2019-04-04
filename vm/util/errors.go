package util

import "errors"

var (
	ErrInvalidMethodParam    = errors.New("invalid method param")
	ErrInsufficientBalance   = errors.New("insufficient balance for transfer")
	ErrAddressCollision      = errors.New("contract address collision")
	ErrIdCollision           = errors.New("id collision")
	ErrExecutionReverted     = errors.New("execution reverted")
	ErrGasUintOverflow       = errors.New("gas uint64 overflow")
	ErrMemSizeOverflow       = errors.New("memory size uint64 overflow")
	ErrReturnDataOutOfBounds = errors.New("vm: return data out of bounds")
	ErrCalcPoWTwice          = errors.New("calc PoW twice referring to one snapshot block")
	ErrAbiMethodNotFound     = errors.New("abi: method not found")
	ErrDepth                 = errors.New("max call depth exceeded")
	ErrInvalidConfirmTime    = errors.New("invalid confirm time")

	ErrAddressNotMatch           = errors.New("current address not match")
	ErrTransactionTypeNotSupport = errors.New("transaction type not supported")
	ErrVersionNotSupport         = errors.New("feature not supported in current snapshot height")
)

func DealWithErr(v interface{}) {
	if v != nil {
		panic(v)
	}
}
