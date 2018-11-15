package walleterrors

import "errors"

var (
	ErrLocked             = errors.New("the crypto store is locked")
	ErrAddressNotFound    = errors.New("not found the given address in the crypto store file")
	ErrInvalidPrikey      = errors.New("invalid prikey")
	ErrDecryptEntropy     = errors.New("error decrypt store")
	ErrEmptyStore         = errors.New("error empty store")
	ErrStoreNotFound      = errors.New("error given store not found ")
	ErrEmptyExtensionWord = errors.New("error empty extension word")
)
