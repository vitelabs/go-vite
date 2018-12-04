package mobile_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/mobile"
	"testing"
)

func TestGetBlocksByAccAddr(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	s, err := client.GetBlocksByAccAddr(a, 0, 20)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}

}

func TestGetAccountByAccAddr(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	s, err := client.GetAccountByAccAddr(a)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}

}
