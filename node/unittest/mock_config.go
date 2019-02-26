package node_unittest

import (
	"github.com/vitelabs/go-vite/config"
	"testing"
)

func MakeMainNetForkPointsConfig() *config.ForkPoints {
	forkPoints := &config.ForkPoints{
		Smart: &config.ForkPoint{
			Height: 5788912,
		},
		Mint: &config.ForkPoint{
			Height: 9473361,
		},
	}

	return forkPoints
}

func MakeTestNetForkPointsConfig() *config.ForkPoints {
	forkPoints := &config.ForkPoints{
		Smart: &config.ForkPoint{
			Height: 500,
		},
		Mint: &config.ForkPoint{
			Height: 1210070,
		},
	}

	return forkPoints
}

func Test(t *testing.T) {

}
