package api

import (
	"encoding/json"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/wallet"
)

func TestDebugApi_ConsensusPlanAndActual(t *testing.T) {
	w := wallet.New(nil)

	unlockAll(w)

	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	vite, err := startVite(w, &addr, t)

	if err != nil {
		panic(err)
	}
	debug := NewDebugApi(vite)
	actual := debug.ConsensusPlanAndActual(types.SNAPSHOT_GID, 0, 0)
	bytes, e := json.Marshal(actual)
	if e != nil {
		panic(e)
	}

	t.Log(string(bytes))
}
