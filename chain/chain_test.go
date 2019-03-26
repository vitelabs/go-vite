package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func NewChainInstance(dirName string, clear bool) (*chain, error) {
	dataDir := path.Join(defaultDataDir(), dirName)
	if clear {
		os.RemoveAll(dataDir)
	}

	chainInstance := NewChain(dataDir)

	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	return chainInstance, nil
}

func defaultDataDir() string {
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

func TestChain_AccountBlock(t *testing.T) {
	chainInstance, err := NewChainInstance("unit_test", true)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("InsertAccountBlock")
	hashList, addrList, heightList := InsertAccountBlock(t, 10000, chainInstance, 10000, 333)
	fmt.Println("Complete InsertAccountBlock")

	fmt.Println("GetAccountBlocksByHash")
	GetAccountBlocksByHash(t, chainInstance, hashList)
	fmt.Println("Complete GetAccountBlocksByHash")

	fmt.Println("GetAccountBlocksByHeight")
	GetAccountBlocksByHeight(t, chainInstance, addrList, heightList)
	fmt.Println("Complete GetAccountBlocksByHeight")

	fmt.Println("IsAccountBlockExisted")
	IsAccountBlockExisted(t, chainInstance, hashList)
	fmt.Println("Complete IsAccountBlockExisted")
}
