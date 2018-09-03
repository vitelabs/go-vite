package vm

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
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

	IsExistGid(gid Gid) bool
}

type RegisterInfo struct {
	data [104]byte
}

func NewRegisterInfo(amount *big.Int, timestamp int64, rewardSnapshotHeight *big.Int) *RegisterInfo {
	r := &RegisterInfo{}
	r.SetAmount(amount)
	r.SetTimestamp(timestamp)
	r.SetRewardSnapshotHeight(rewardSnapshotHeight)
	return r
}

func NewRegisterInfoFromBytes(data []byte) *RegisterInfo {
	r := &RegisterInfo{}
	copy(r.data[:], rightPadBytes(data, 104)[0:104])
	return r
}

func (r *RegisterInfo) Amount() *big.Int {
	return new(big.Int).SetBytes(r.data[0:32])
}
func (r *RegisterInfo) SetAmount(amount *big.Int) {
	copy(r.data[0:32], leftPadBytes(amount.Bytes(), 32))
}
func (r *RegisterInfo) Timestamp() int64 {
	return new(big.Int).SetBytes(r.data[32:40]).Int64()
}
func (r *RegisterInfo) SetTimestamp(timestamp int64) {
	copy(r.data[32:40], leftPadBytes(big.NewInt(timestamp).Bytes(), 8))
}
func (r *RegisterInfo) RewardSnapshotHeight() *big.Int {
	return new(big.Int).SetBytes(r.data[40:72])
}
func (r *RegisterInfo) SetRewardSnapshotHeight(rewardSnapshotHeight *big.Int) {
	copy(r.data[40:72], leftPadBytes(rewardSnapshotHeight.Bytes(), 32))
}
func (r *RegisterInfo) CancelSnapshotHeight() *big.Int {
	return new(big.Int).SetBytes(r.data[72:104])
}
func (r *RegisterInfo) SetCancelSnapshotHeight(cancelSnapshotHeight *big.Int) {
	copy(r.data[72:104], leftPadBytes(cancelSnapshotHeight.Bytes(), 32))
}

// TODO tmp
type Gid [10]byte

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
