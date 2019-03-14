package config

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/vitelabs/go-vite/config/biz"
)

type Config struct {
	*Producer   `json:"Producer"`
	*Chain      `json:"Chain"`
	*Vm         `json:"Vm"`
	*Subscribe  `json:"Subscribe"`
	*Net        `json:"Net"`
	*biz.Reward `json:"Reward"`
	*Genesis    `json:"Genesis"`

	// global keys
	DataDir string `json:"DataDir"`
	//Log level
	LogLevel string `json:"LogLevel"`
}

func (c Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog")
}

// DefaultDataDir is the default data directory to use for the databases and other persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "GVite")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "GVite")
		} else {
			return filepath.Join(home, ".gvite")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
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
