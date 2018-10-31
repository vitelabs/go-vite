package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow/remote"
	"testing"
)

func TestPow_GetPowNonce(t *testing.T) {
	p := Pow{}
	remote.InitUrl("127.0.0.1")
	addr, _, _ := types.CreateAddress()
	hash := types.ZERO_HASH
	dataHash := types.DataListHash(addr.Bytes(), hash.Bytes())
	nonce := p.GetPowNonce("", dataHash)
	fmt.Printf("Nonce:%v\n", nonce)
}
