package main

import (
	"bufio"
	"fmt"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"os"
)

func main() {
	kc := keystore.DefaultKeyConfig
	km := keystore.NewManager(&kc)
	km.Init()
	printStatus(km)

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter any key to stop ")
	input, err := inputReader.ReadString('\n')
	if err == nil {
		fmt.Printf("The input was: %s\n", input)
	}

}

func printStatus(km *keystore.Manager) {
	s, _ := km.Status()
	println(s)
}
