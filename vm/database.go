package vm

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

type Log struct {
	// list of topics provided by the contract
	Topics []types.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
}

type VmDatabase interface {
	Balance(addr types.Address, tokenId types.TokenTypeId) *big.Int
	SubBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)
	AddBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int)

	SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock
	SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock
	// forward=true return (startHeight, startHeight+count], forward=false return [startHeight-count, start)
	SnapshotBlockList(startHeight *big.Int, count uint64, forward bool) []VmSnapshotBlock

	AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock

	Rollback()

	IsExistAddress(addr types.Address) bool

	IsExistToken(tokenId types.TokenTypeId) bool
	CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool

	SetContractGid(addr types.Address, gid Gid, open bool)
	SetContractCode(addr types.Address, gid Gid, code []byte)
	ContractCode(addr types.Address) []byte

	Storage(addr types.Address, loc types.Hash) []byte
	SetStorage(addr types.Address, loc types.Hash, value []byte)
	PrintStorage(addr types.Address) string
	StorageHash(addr types.Address) types.Hash

	AddLog(*Log)
	LogListHash() types.Hash
}

// TODO tmp
type Gid [10]byte

func DataToGid(data ...[]byte) Gid {
	gid, _ := BytesToGid(crypto.Hash(10, data...))
	return gid
}

func BigToGid(data *big.Int) (Gid, error) {
	return BytesToGid(LeftPadBytes(data.Bytes(), 10))
}
func BytesToGid(b []byte) (Gid, error) {
	var gid Gid
	err := gid.SetBytes(b)
	return gid, err
}
func (gid *Gid) SetBytes(b []byte) error {
	if len(b) != 10 {
		return fmt.Errorf("error hash size %v", len(b))
	}
	copy(gid[:], b)
	return nil
}
func (gid *Gid) Bytes() []byte {
	return gid[:]
}
