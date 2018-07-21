package main

import (
	"crypto/rand"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"fmt"
	"github.com/vitelabs/go-vite/vite"
	"log"
	"github.com/vitelabs/go-vite/p2p"
)

func main()  {
	v, err := vite.New(&p2p.Config{})
	if err != nil {
		log.Fatal(err)
	}
	mockAccount(v)
}


func mockAccount (v *vite.Vite) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	addr := types.PubkeyToAddress(publicKey)
	fmt.Println("Current AccountAddress: ", addr.Hex())
	fmt.Println("Current PublicKey: ", publicKey.Hex())
	fmt.Println("Current PrivateKey: ", privateKey.Hex())

	go func() {

	}()
}