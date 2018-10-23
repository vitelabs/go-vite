package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestChain_GetAccount(t *testing.T) {
	chainInstance := getChainInstance()
	addr, _ := types.HexToAddress("vite_5acd0b2ef651bdc0c586aafe7a780103f45ac532cd886eb859")
	account, _ := chainInstance.GetAccount(&addr)
	fmt.Printf("%+v\n", account)
}
