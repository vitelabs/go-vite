package jsre

import (
	"testing"

	"github.com/vitelabs/go-vite/v2/cmd/internal/jsre/deps"
)

func TestAssetData(t *testing.T) {
	deps.MustAsset("polyfill.js")
	deps.MustAsset("vite.js")
	deps.MustAsset("docs.js")
}
