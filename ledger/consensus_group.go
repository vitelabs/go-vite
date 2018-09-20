package ledger

import "github.com/vitelabs/go-vite/common/types"

var commonGid = types.Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

func CommonGid() types.Gid {
	return commonGid
}
