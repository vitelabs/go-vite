package walleterrors

import "errors"

var (
	ErrLocked        = errors.New("the seed store is locked")
	ErrNotFind       = errors.New("not found the given address in the seed store file")
	ErrInvalidPrikey = errors.New("invalid prikey")
	ErrAlreadyLocked = errors.New("the address was previously unlocked")
	ErrDecryptSeed   = errors.New("error decrypting seed")
)
