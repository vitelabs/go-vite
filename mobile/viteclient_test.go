package mobile_test

import (
	"testing"
	"github.com/vitelabs/go-vite/mobile"
	"fmt"
)

func TestDial(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")
	s, _ := client.GetBlocksByAccAddr(a, 0, 20)
	fmt.Println(s)
}
