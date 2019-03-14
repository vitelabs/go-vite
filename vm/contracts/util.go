package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_db"
)

func IsExistGid(db vm_db.VMDB, gid types.Gid) bool {
	value := db.GetValue(abi.GetConsensusGroupKey(gid))
	return len(value) > 0
}
