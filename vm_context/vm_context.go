package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type VmContext struct {
	currentSnapshotBlockHash *types.Hash
	prevAccountBlockHash     *types.Hash
	address                  *types.Address
}

func NewVmContext(snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) *VmContext {
	return &VmContext{
		currentSnapshotBlockHash: snapshotBlockHash,
		prevAccountBlockHash:     prevAccountBlockHash,
		address:                  addr,
	}
}
func GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	return big.NewInt(0)
}

func SubBalance(addr *types.Address, tokenTypeId *types.TokenTypeId, amount *big.Int) error {
	return nil
}

func AddBalance(addr *types.Address, tokenTypeId *types.TokenTypeId, amount *big.Int) error {
	return nil
}

func GetSnapshotBlock(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func GetSnapshotBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func Reset() {

}
func SetContractCode(addr *types.Address, gid types.Gid, code []byte) {

}

func GetContractCode(addr *types.Address, gid types.Gid, code []byte) {

}

func SetToken() {

}

func GetToken(id *types.TokenTypeId) {

}

func SetStorage(key []byte, value []byte) {

}

func GetStorage(key []byte) {

}

func GetStorageHash() *types.Hash {

}

func AddLog() {

}

func GetLogListHash() {

}
func IsAddressExisted() {

}

func GetAccoutBlockByHash() {

}
