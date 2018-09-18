package vmctxt_interface

import "github.com/vitelabs/go-vite/common/types"

type ContractGid interface {
	Gid() *types.Gid
	Addr() *types.Address
}
