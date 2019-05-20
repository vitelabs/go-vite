package config_gen

import (
	"github.com/vitelabs/go-vite/config"
	"testing"
)

func TestMakeGenesisAccountConfig(t *testing.T) {
	cfg := makeGenesisAccountConfig()
	if !config.IsCompleteGenesisConfig(cfg) {
		t.Fatalf("convert genesis config failed")
	}
}
