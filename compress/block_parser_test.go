package compress

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"os"
	"testing"
)

func Test_BlockParser(t *testing.T) {
	file, err := os.Open("./subgraph_1_3600")
	if err != nil {
		t.Fatal(err)
	}

	BlockParser(file, 0, func(block ledger.Block, err error) {
		switch block.(type) {
		case *ledger.AccountBlock:
			b := block.(*ledger.AccountBlock)
			if b.Hash.String() == "f9c014a350e9a9b4d0d37e7f29038c91842d1c509f586b09c12972a4a9f5f392" {
				fmt.Printf("%+v\n", block)
			}
		}

	})
}
