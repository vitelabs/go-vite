package main

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"time"
)

func createAddress () *types.Address {
	rand.Seed(time.Time{}.Unix())
	addressBytes := []byte{}
	for i:=0 ; i < 20; i ++ {
		addressBytes = append(addressBytes, byte(rand.Intn(256)))
	}

	address, _:= types.BytesToAddress(addressBytes)

	return &address
}
