package ledgererrors

import "errors"

var (
	ErrBalanceNotEnough = errors.New("The balance is not enough.")
)
