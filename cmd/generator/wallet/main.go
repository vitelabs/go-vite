package main

import (
	"flag"
	"fmt"

	"github.com/tyler-smith/go-bip39"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
)

var num = flag.Int("num", 1, "num ")

func main() {
	flag.Parse()
	for i := 0; i < *num; i++ {
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			panic(err)
		}
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			panic(err)
		}
		seed := bip39.NewSeed(mnemonic, "")
		key, err := derivation.DeriveWithIndex(0, seed)
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
