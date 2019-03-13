package ledger

import "github.com/vitelabs/go-vite/common/types"

type ContractMeta struct {
	SendConfirmedTimes uint8
	Gid                *types.Gid
}
