package api

import "errors"

var (
	ErrStrToBigInt                    = errors.New("convert to big.Int failed")
	ErrPoWNotSupportedUnderCongestion = errors.New("PoW service not supported")
)
