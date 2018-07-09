package keystore

import (
	"github.com/vitelabs/go-vite/common"
	"path/filepath"
)

type KeyConfig struct {
	KeyStoreDir       string
	UseLightweightKDF bool
}

var DefaultKeyConfig = KeyConfig{
	KeyStoreDir:       common.DefaultDataDir(),
	UseLightweightKDF: false,
}

var TestKeyConfig = KeyConfig{
	KeyStoreDir:       filepath.Join(common.TestDataDir(), "wallet"),
	UseLightweightKDF: false,
}
