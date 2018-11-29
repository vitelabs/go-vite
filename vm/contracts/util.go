package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

func IsUserAccount(db vmctxt_interface.VmDatabase, addr types.Address) bool {
	return len(db.GetContractCode(&addr)) == 0
}

func IsExistGid(db vmctxt_interface.VmDatabase, gid types.Gid) bool {
	value := db.GetStorage(&types.AddressConsensusGroup, abi.GetConsensusGroupKey(gid))
	return len(value) > 0
}

// TODO following method belongs to vm
func GetCreateContractData(bytecode []byte, gid types.Gid) []byte {
	return append(gid.Bytes(), bytecode...)
}

func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}

func NewContractAddress(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}
