package walleterrors

import "errors"

var (
	ErrLocked        = errors.New("need password or unlock")
	ErrNotFind       = errors.New("not found the given address in any keystore file")
	ErrInvalidPrikey = errors.New("invalid prikey")
	ErrAlreadyLocked = errors.New("the address was previously unlocked")
	ErrDecryptKey    = errors.New("error decrypting key")
)
