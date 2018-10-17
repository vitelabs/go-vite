package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
	"testing"
)

var (
	addrStr = "vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1"
	addr, _ = types.HexToAddress(addrStr)
)

func PrepareVite() (*vite.Vite, error) {
	config := &config.Config{
		DataDir: common.DefaultDataDir(),
		Producer: &config.Producer{
			Producer: true,
			Coinbase: addr.String(),
		},
		Vm: &config.Vm{IsVmTest: false},
		Net: &config.Net{
			Single: true,
		},
	}
	w := wallet.New(nil)
	vite, err := vite.New(config, w)
	if err != nil {
		return nil, err
	}

	return vite, nil
}

func TestLedgerApi_GetBlocksByAccAddr(t *testing.T) {
	vite, err := PrepareVite()
	if err != nil {
		t.Error(err)
		return
	}
	lApi := NewLedgerApi(vite)

	if err != nil {
		t.Error(err)
		return
	}

	list, err := lApi.GetBlocksByAccAddr(addr, 0, 100)
	if err != nil {
		t.Error(err)
		return
	}

	for _, b := range list {
		fmt.Printf("delete %+v\n", b)
	}

}
