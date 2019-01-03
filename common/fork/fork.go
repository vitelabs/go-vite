package fork

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
)

var forkPoints config.ForkPoints

func SetForkPoints(points *config.ForkPoints) {
	forkPoints = *points
}

func IsVite1(blockHeight uint64) bool {
	return forkPoints.Vite1.Height > 0 && blockHeight >= forkPoints.Vite1.Height
}

func IsVite1ByHash(blockHash types.Hash) bool {
	return forkPoints.Vite1.Hash != nil && blockHash == *forkPoints.Vite1.Hash
}
