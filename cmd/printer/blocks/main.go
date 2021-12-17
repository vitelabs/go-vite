package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/helper"
	chain_block "github.com/vitelabs/go-vite/v2/ledger/chain/block"
	chain_file_manager "github.com/vitelabs/go-vite/v2/ledger/chain/file_manager"
)

// env GOOS=linux GOARCH=amd64 go build -i -o blocks_printer ./cmd/printer/blocks/main.go
var fieldId = flag.Uint64("fieldId", 1, "filed id")
var offset = flag.Int64("offset", 0, "offset")

var dir = flag.String("dataDir", "devdata", "data dir, for example: devdata")

func main() {
	flag.Parse()
	db, err := chain_block.NewBlockDB(*dir)
	helper.AssertNil(err)
	location := &chain_file_manager.Location{
		FileId: *fieldId,
		Offset: *offset,
	}
	sb, ab, next, err := db.ReadUnit(location)
	helper.AssertNil(err)
	fmt.Println("next location", next.FileId, next.Offset)
	if ab != nil {
		fmt.Println("account block", ab.AccountAddress, ab.Height, ab.Hash)
		byt, _ := json.Marshal(ab)
		fmt.Println(string(byt))
	}
	if sb != nil {
		fmt.Println("snapshot block", sb.Height, sb.Hash)
		byt, _ := json.Marshal(sb)
		fmt.Println(string(byt))
	}
}
