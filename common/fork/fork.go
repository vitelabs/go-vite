package fork

import (
	"github.com/vitelabs/go-vite/config"
)

var forkPoints config.ForkPoints

func SetForkPoints(points *config.ForkPoints) {
	forkPoints = *points
}

func IsVite1(blockHeight uint64) bool {
	return forkPoints.Vite1.Height > 0 && blockHeight >= forkPoints.Vite1.Height
}

func GetForkPoints() config.ForkPoints {
	return forkPoints
}
