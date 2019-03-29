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

func TestChain(t *testing.T) {

	const accountNum = 1000
	chainInstance, err := NewChainInstance("unit_test", false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("InsertAccountBlock")

	accounts, hashList, addrList, heightList, snapshotBlockList := InsertAccountBlock(t, accountNum, chainInstance, 1000, 198)

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
	//if maxAccountId > accountNum {
	//	t.Fatal("error!")
	//}

	fmt.Println("Complete InsertAccountBlock")

	t.Run("GetAccountBlockByHash", func(t *testing.T) {
		GetAccountBlockByHash(t, chainInstance, hashList)
	})

	t.Run("GetAccountBlockByHeight", func(t *testing.T) {
		GetAccountBlockByHeight(t, chainInstance, addrList, heightList)
	})

	t.Run("IsAccountBlockExisted", func(t *testing.T) {
		IsAccountBlockExisted(t, chainInstance, hashList)
	})

	t.Run("GetAccountId", func(t *testing.T) {
		GetAccountId(t, chainInstance, addrList, accountIdList)
	})

	t.Run("GetAccountAddress", func(t *testing.T) {
		GetAccountAddress(t, chainInstance, addrList, accountIdList)
	})

	t.Run("IsReceived", func(t *testing.T) {
		IsReceived(t, chainInstance, accounts, hashList)
	})

	t.Run("GetReceiveAbBySendAb", func(t *testing.T) {
		GetReceiveAbBySendAb(t, chainInstance, accounts, hashList)
	})

	t.Run("GetConfirmedTimes", func(t *testing.T) {
		GetConfirmedTimes(t, chainInstance, accounts, hashList)
	})

	t.Run("GetLatestAccountBlock", func(t *testing.T) {
		GetLatestAccountBlock(t, chainInstance, accounts, addrList)
	})

	t.Run("GetLatestAccountHeight", func(t *testing.T) {
		GetLatestAccountHeight(t, chainInstance, accounts, addrList)
	})

	t.Run("HasOnRoadBlocks", func(t *testing.T) {
		HasOnRoadBlocks(t, chainInstance, accounts, addrList)
	})

	t.Run("GetOnRoadBlocksHashList", func(t *testing.T) {
		GetOnRoadBlocksHashList(t, chainInstance, accounts, addrList)
	})

	t.Run("IsSnapshotBlockExisted", func(t *testing.T) {
		IsSnapshotBlockExisted(t, chainInstance, snapshotBlockList)
	})
	t.Run("GetGenesisSnapshotBlock", func(t *testing.T) {
		GetGenesisSnapshotBlock(t, chainInstance)
	})
	t.Run("GetLatestSnapshotBlock", func(t *testing.T) {
		GetLatestSnapshotBlock(t, chainInstance)
	})
	t.Run("GetSnapshotHeightByHash", func(t *testing.T) {
		GetSnapshotHeightByHash(t, chainInstance, snapshotBlockList)
	})
	t.Run("GetSnapshotHeaderByHeight", func(t *testing.T) {
		GetSnapshotHeaderByHeight(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetSnapshotBlockByHeight", func(t *testing.T) {
		GetSnapshotBlockByHeight(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetSnapshotHeaderByHash", func(t *testing.T) {
		GetSnapshotHeaderByHash(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetSnapshotBlockByHash", func(t *testing.T) {
		GetSnapshotBlockByHash(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetRangeSnapshotHeaders", func(t *testing.T) {
		GetRangeSnapshotHeaders(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetRangeSnapshotBlocks", func(t *testing.T) {
		GetRangeSnapshotBlocks(t, chainInstance, snapshotBlockList)
	})

	t.Run("GetBalance", func(t *testing.T) {
		GetBalance(t, chainInstance, accounts)
	})

	t.Run("GetBalanceMap", func(t *testing.T) {
		GetBalanceMap(t, chainInstance, accounts)
	})

	t.Run("GetAccountBlocks", func(t *testing.T) {
		GetAccountBlocks(t, chainInstance, accounts, addrList)
	})

	t.Run("GetAccountBlocksByHeight", func(t *testing.T) {
		GetAccountBlocksByHeight(t, chainInstance, accounts, addrList)
	})
}
