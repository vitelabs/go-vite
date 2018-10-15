package jsre

import (
	"github.com/vitelabs/go-vite/cmd/internal/jsre/deps"
	"testing"
)

func TestAssetData(t *testing.T) {
	deps.MustAsset("bignumber.js")
	deps.MustAsset("vite.js")
	deps.MustAsset("typedarray.js")
}
