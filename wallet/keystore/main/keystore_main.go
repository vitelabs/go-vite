package main

import (
	"bufio"
	"fmt"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"os"
)

func main() {

	kc := wallet.TestKeyConfig
	km := keystore.NewManager(&kc)
	km.Init()

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter any key to stop ")
	input, err := inputReader.ReadString('\n')
	if err == nil {
		fmt.Printf("The input was: %s\n", input)
	}

}
