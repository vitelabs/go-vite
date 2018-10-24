package walleterrors

import "errors"

var (
	ErrLocked         = errors.New("the crypto store is locked")
	ErrNotFind        = errors.New("not found the given address in the crypto store file")
	ErrInvalidPrikey  = errors.New("invalid prikey")
	ErrAlreadyLocked  = errors.New("the address was previously unlocked")
	ErrDecryptEntropy = errors.New("error decrypt store")
)
