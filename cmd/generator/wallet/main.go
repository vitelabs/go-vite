package main

import (
	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/wallet"
)

var num = flag.Int("num", 1, "num ")
var dir = flag.String("dataDir", "wallet", "data dir, for example: devdata")
var passwd = flag.String("passwd", "123456", "default passwd is 123456")

func main() {
	flag.Parse()
	manager := wallet.New(&wallet.Config{
		DataDir: *dir,
	})
	for i := 0; i < *num; i++ {
		mnemonic, em, err := manager.NewMnemonicAndEntropyStore(*passwd)
		if err != nil {
			panic(err)
		}
		err = em.Unlock(*passwd)
		if err != nil {
			panic(err)
		}
		_, key, err := em.DeriveForIndexPath(uint32(0))
		if err != nil {
			panic(err)
		}
		address, err := key.Address()
		if err != nil {
			panic(err)
		}
		priKeys, err := key.PrivateKey()
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s,%s,%s\n", address, priKeys.Hex(), mnemonic)
	}
}
