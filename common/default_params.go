package common

import (
	"path/filepath"
	"os"
	"os/user"
)

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
