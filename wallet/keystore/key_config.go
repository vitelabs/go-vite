package keystore

import (
	"github.com/vitelabs/go-vite/common"
)

type KeyConfig struct {
	KeyStoreDir       string
	UseLightweightKDF bool
}

var DefaultKeyConfig = KeyConfig{
	KeyStoreDir:       common.DefaultDataDir(),
	UseLightweightKDF: false,
}
