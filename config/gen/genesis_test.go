package config_gen

import (
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto"
	"gotest.tools/assert"
)

func TestMakeGenesisAccountConfig(t *testing.T) {
	cfg := makeGenesisAccountConfig()
	if !config.IsCompleteGenesisConfig(cfg) {
		t.Fatalf("convert genesis config failed")
	}
}

func TestGenesisCfg(t *testing.T) {
	hash, err := types.BytesToHash(crypto.Hash256([]byte(genesisJsonStr)))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hash)
	assert.Equal(t, "445e457d30a15a74da9989f5cb989221c048ba230f069bb43b6668f56c3b3533", hash.String())
}
