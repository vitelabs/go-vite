package main

import (
	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/wallet"
)

var num = flag.Int("num", 1, "num ")

func main() {
	flag.Parse()
	for i := 0; i < *num; i++ {
		addr, key, mnemonic, err := wallet.RandomMnemonic24()
		if err != nil {
			panic(err)
		}
		fmt.Printf("address:%s, key:%s, mnemonic:%s\n", addr, key.Hex(), mnemonic)
	}
}
