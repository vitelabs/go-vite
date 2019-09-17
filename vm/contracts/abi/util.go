package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

// StorageDatabase define db interfaces for querying built-in contract
type StorageDatabase interface {
	GetValue(key []byte) ([]byte, error)
	NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error)
	Address() *types.Address
}

func filterKeyValue(key, value []byte, f func(key []byte) bool) bool {
	if len(value) > 0 && (f == nil || f(key)) {
		return true
	}
	return false
}
