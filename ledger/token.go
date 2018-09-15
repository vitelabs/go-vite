package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
)

func ViteTokenId() *types.TokenTypeId {
	return &types.TokenTypeId{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
}
