package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	Retry   = true
	NoRetry = false
)

func IsViteToken(tokenId types.TokenTypeId) bool {
	return tokenId == ledger.ViteTokenId
}
func IsSnapshotGid(gid types.Gid) bool {
	return gid == types.SNAPSHOT_GID
}
