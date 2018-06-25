package account

import (
	"os"
	"os/user"
	"path/filepath"
)

type KeyConfig struct {
	KeyStoreDir       string
	UseLightweightKDF bool
}

var DefaultKeyConfig = KeyConfig{
	KeyStoreDir:       DefaultDataDir(),
	UseLightweightKDF: false,
}

func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		return filepath.Join(home, ".viteisbest")
	}
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
