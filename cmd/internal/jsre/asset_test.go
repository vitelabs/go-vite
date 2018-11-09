package jsre

import (
	"testing"

	"github.com/vitelabs/go-vite/cmd/internal/jsre/deps"
)

func TestAssetData(t *testing.T) {
	deps.MustAsset("polyfill.js")
	deps.MustAsset("vite.js")
	deps.MustAsset("docs.js")
}
