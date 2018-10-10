package vite

import (
	"flag"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

var genesisAccountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)

}

func PrepareVite() (chain.Chain, *generator.Generator, *verifier.AccountVerifier) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)
	g := generator.NewGenerator(c, w.KeystoreManager)

	v := verifier.NewAccountVerifier(c, nil)

	return c, g, v
}

func TestSend(t *testing.T) {

}
