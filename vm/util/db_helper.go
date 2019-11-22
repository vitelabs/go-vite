package util

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type dbInterface interface {
	GetBalance(tokenTypeID *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeID *types.TokenTypeId, amount *big.Int)
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) error

	LatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	Address() *types.Address
	IsContractAccount() (bool, error)
	GetContractCode() ([]byte, error)
	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error)
}

// AddBalance add balance of certain token to current account.
// Used in response block.
// Panics when db error.
func AddBalance(db dbInterface, id *types.TokenTypeId, amount *big.Int) {
	b, err := db.GetBalance(id)
	DealWithErr(err)
	b.Add(b, amount)
	db.SetBalance(id, b)
}

// SubBalance sub balance of certain token to current account.
// Balance check must be performed before calling this method.
// Used in request block.
// Panics when db error.
func SubBalance(db dbInterface, id *types.TokenTypeId, amount *big.Int) {
	b, err := db.GetBalance(id)
	DealWithErr(err)
	if b.Cmp(amount) >= 0 {
		b.Sub(b, amount)
		db.SetBalance(id, b)
	}
}

// GetValue get current contract storage value by key.
// Panics when db error.
func GetValue(db dbInterface, key []byte) []byte {
	v, err := db.GetValue(key)
	DealWithErr(err)
	return v
}

// SetValue update current contract storage value by key.
// Delete key when value is nil.
// Key len must be no more than 32 bytes.
// Panics when db error.
func SetValue(db dbInterface, key []byte, value []byte) {
	err := db.SetValue(key, value)
	DealWithErr(err)
}

// GetContractCode get contract in certain snapshot block.
// if input addr == current address,
//   return contract code in latest contract account block.
// if input addr != current address,
//   return contract code in snapshot block of global status.
// Panics when db error.
func GetContractCode(db dbInterface, addr *types.Address, status GlobalStatus) ([]byte, []byte) {
	var code []byte
	var err error
	if *db.Address() == *addr {
		code, err = db.GetContractCode()
	} else {
		code, err = db.GetContractCodeBySnapshotBlock(addr, status.SnapshotBlock())
	}
	DealWithErr(err)
	if len(code) > 0 {
		return code[:contractTypeSize], code[contractTypeSize:]
	}
	return nil, nil
}
