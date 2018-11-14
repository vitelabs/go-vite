package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/wallet/hd-bip/derivation"
)

func main() {
	//if err := derivation.RandomMnemonic12(""); err != nil {
	//	fmt.Println(err)
	//}
	//
	//if err := derivation.RandomMnemonic12("123456"); err != nil {
	//	fmt.Println(err)
	//}
	//
	//var b [16]byte
	//if err := derivation.Menmonic(b[:],""); err != nil {
	//	fmt.Println(err)
	//}

	if err := derivation.RandomMnemonic15(""); err != nil {
		fmt.Println(err)
	}

	//if err := derivation.RandomMnemonic24("123456"); err != nil {
	//	fmt.Println(err)
	//}

}
