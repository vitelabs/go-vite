package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto"
)

func TestMakeGenesisAccountConfig(t *testing.T) {
	cfg := MainnetGenesis()
	if !IsCompleteGenesisConfig(cfg) {
		t.Fatalf("convert genesis config failed")
	}
}

func TestGenesisCfg(t *testing.T) {
	hash, err := types.BytesToHash(crypto.Hash256([]byte(GenesisJson())))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hash)
	assert.Equal(t, "4c56bfcc9a28d902352c3d17fcd9de147538ae9f7f71aff975433b8d403e45df", hash.String())
}

func TestLoadGenesisCfg(t *testing.T) {
	loadFromGenesisFile("~/go/src/github.com/vitelabs/go-vite/conf/evm/genesis.json")
}
