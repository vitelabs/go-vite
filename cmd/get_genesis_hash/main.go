package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
)

func main() {
	fmt.Println(ledger.GetSnapshotGenesisBlock().ComputeHash())
}
