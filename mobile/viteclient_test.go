package mobile_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/mobile"
	"testing"
)

func TestDial(t *testing.T) {
	client, _ := mobile.Dial("http://127.0.0.1:48132")
	a, _ := mobile.NewAddressFromString("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")
	s, err := client.GetBlocksByAccAddr(a, 0, 20)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}
	client.Ttt()
}
