package common

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultHTTPHost = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 48132       // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort   = 31420       // Default TCP port for the websocket RPC server
	DefaultP2PPort  = 8483
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

func DefaultHttpEndpoint() string {
	return ":48132"
}

func DefaultWSEndpoint() string {
	return ":31420"
}

func DefaultIpcFile() string {
	endpoint := "vite.ipc"
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\vite.ipc`
	}
	return endpoint
}

func MakeDefaultLogger(absFilePath string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   absFilePath,
		MaxSize:    100,
		MaxBackups: 100000,
		MaxAge:     100000,
		Compress:   true,
		LocalTime:  true,
	}
}
