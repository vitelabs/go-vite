package node

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/vitelabs/go-vite/common"
)

var DefaultNodeConfig = Config{
	Name:              "nodeServer",
	IPCPath:           "gvite.ipc",
	DataDir:           DefaultDataDir(),
	KeyStoreDir:       DefaultDataDir(),
	HttpPort:          common.DefaultHTTPPort,
	WSPort:            common.DefaultWSPort,
	Discover:          true,
	LogLevel:          "info",
	WSOrigins:         []string{"*"},
	WSExposeAll:       true,
	HttpExposeAll:     true,
	FileListenAddress: "0.0.0.0:8484",
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
	// As we cannot guess chain stable location, return empty and handle later
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
