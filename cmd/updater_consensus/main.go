package main

import (
	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/consensus/cdb"
)

var hash = flag.String("hash", "", "hash")
var dir = flag.String("dataDir", "", "consensus data dir")

func main() {
	flag.Parse()

	if *dir == "" {
		panic("err dir")
	}
	d, err := leveldb.OpenFile(*dir, nil)
	if err != nil {
		panic(err)
	}

	db := cdb.NewConsensusDB(d)
	addrList, err := db.GetElectionResultByHash(types.HexToHashPanic(*hash))
	if err != nil {
		panic(err)
	}
	for k, v := range addrList {
		fmt.Println(k, v)
	}
	err = db.DeleteElectionResultByHash(types.HexToHashPanic(*hash))
	if err != nil {
		panic(err)
	}
}
