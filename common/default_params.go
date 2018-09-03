package common

import (
	"path/filepath"
	"os"
	"os/user"
	"runtime"
)

// DefaultDataDir is  $HOME/viteisbest/
func DefaultDataDir() string {
	home := HomeDir()
	if home != "" {
		return filepath.Join(home, "viteisbest")
	}
	return ""
}

//it is the dir in go-vite/testdata
func GoViteTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filepath.Dir(filename)), "testdata")
}

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func DefaultWSEndpoint() string {
	return "192.168.31.235:31420"
}

func DefaultIpcFile() string {
	endpoint := "vite.ipc"
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\vite.ipc`
	}
	return endpoint
}