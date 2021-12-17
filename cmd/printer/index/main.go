package main

import (
	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	chain_index "github.com/vitelabs/go-vite/v2/ledger/chain/index"
)

//  env GOOS=linux GOARCH=amd64 go build -i -o index_printer ./cmd/printer/index/main.go

var hash = flag.String("hash", "", "hash")
var dir = flag.String("dataDir", "devdata", "data dir, for example: devdata")

func main() {
	flag.Parse()
	db, err := chain_index.NewIndexDB(*dir)

	helper.AssertNil(err)
	hash := types.HexToHashPanic(*hash)
	location, err := db.GetAccountBlockLocationByHash(&hash)
	helper.AssertNil(err)
	fmt.Println("location", location.FileId, location.Offset)
}
