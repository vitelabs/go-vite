package walleterrors

import "errors"

var (
	ErrLocked        = errors.New("the address is locked")
	ErrNotFind       = errors.New("not found the given address in any keystore file")
	ErrInvalidPrikey = errors.New("invalid prikey")
	ErrAlreadyLocked = errors.New("the address was previously unlocked")
	ErrDecryptKey    = errors.New("error decrypting key")
)
