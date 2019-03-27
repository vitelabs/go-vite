package chain

import (
	"fmt"
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

	const accountNum = 1000
	chainInstance, err := NewChainInstance("unit_test", true)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("InsertAccountBlock")
	accounts, hashList, addrList, heightList := InsertAccountBlock(t, accountNum, chainInstance, 10000, 198)

	accountIdList := make([]uint64, len(addrList))
	maxAccountId := uint64(0)
	for index, addr := range addrList {
		accountId, err := chainInstance.GetAccountId(addr)
		if err != nil {
			t.Fatal(err)
		}
		if accountId <= 0 {
			t.Fatal("accountId <= 0")
		} else if accountId > maxAccountId {
			maxAccountId = accountId
		}

		accountIdList[index] = accountId
	}
	if maxAccountId > accountNum {
		t.Fatal("error!")
	}

	fmt.Println("Complete InsertAccountBlock")

	fmt.Println("GetAccountBlockByHash")
	GetAccountBlockByHash(t, chainInstance, hashList)
	fmt.Println("Complete GetAccountBlockByHash")

	fmt.Println("GetAccountBlockByHeight")
	GetAccountBlockByHeight(t, chainInstance, addrList, heightList)
	fmt.Println("Complete GetAccountBlockByHeight")

	fmt.Println("IsAccountBlockExisted")
	IsAccountBlockExisted(t, chainInstance, hashList)
	fmt.Println("Complete IsAccountBlockExisted")

	fmt.Println("GetAccountId")
	GetAccountId(t, chainInstance, addrList, accountIdList)
	fmt.Println("Complete GetAccountId")

	fmt.Println("IsReceived")
	IsReceived(t, chainInstance, accounts, hashList)
	fmt.Println("Complete IsReceived")

	fmt.Println("GetReceiveAbBySendAb")
	GetReceiveAbBySendAb(t, chainInstance, accounts, hashList)
	fmt.Println("Complete GetReceiveAbBySendAb")

	fmt.Println("GetAccountAddress")
	GetAccountAddress(t, chainInstance, addrList, accountIdList)
	fmt.Println("Complete GetAccountAddress")

	fmt.Println("GetConfirmedTimes")
	GetConfirmedTimes(t, chainInstance, accounts, hashList)
	fmt.Println("GetConfirmedTimes")

	fmt.Println("GetLatestAccountBlock")
	GetLatestAccountBlock(t, chainInstance, accounts, addrList)
	fmt.Println("Complete GetLatestAccountBlock")

	fmt.Println("GetLatestAccountHeight")
	GetLatestAccountHeight(t, chainInstance, accounts, addrList)
	fmt.Println("Complete GetLatestAccountHeight")

	fmt.Println("GetAccountBlocks")
	GetAccountBlocks(t, chainInstance, accounts, addrList)
	fmt.Println("Complete GetAccountBlocks")

	fmt.Println("GetAccountBlocksByHeight")
	GetAccountBlocksByHeight(t, chainInstance, accounts, addrList)
	fmt.Println("Complete GetAccountBlocksByHeight")

}
