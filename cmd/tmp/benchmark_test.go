package main

import (
	"path/filepath"
	"testing"

	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet"
)

var passwd = "123456"

func TestBenchmark(t *testing.T) {
	baseDir := filepath.Join(common.DefaultDataDir(), "benchmark")

	cfg := &wallet.Config{DataDir: filepath.Join(baseDir, "wallet")}
	w := wallet.New(cfg)

	genesis, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	vite, _, err := startNode(w, &genesis, filepath.Join(baseDir, "chain"))
	if err != nil {
		panic(err)
	}
	vite.OnRoad().StartAutoReceiveWorker(genesis, nil)

	for printBalance(vite, genesis).Sign() == 0 {
		time.Sleep(time.Second)
	}

	var addrList []types.Address
	N := 5
	for i := 0; i < N; i++ {
		addr, keys, _ := types.CreateAddress()
		w.KeystoreManager.ImportPriv(keys.Hex(), passwd)
		addrList = append(addrList, addr)
	}

}
