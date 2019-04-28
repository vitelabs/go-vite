package main

import (
	"flag"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/db"
)

var hash = flag.String("hash", "", "remove consensus result hash")
var dir = flag.String("dataDir", "devdata", "data dir, for example: devdata")
var delete = flag.Bool("delete", false, "delete ?")

func main() {
	flag.Parse()
	db, err := leveldb.OpenFile(*dir, nil)

	if err != nil {
		panic(err)
	}

	hashPanic := types.HexToHashPanic(*hash)

	cDB := consensus_db.NewConsensusDB(db)
	addr, err := cDB.GetElectionResultByHash(hashPanic)
	if err != nil {
		panic(err)
	}

	fmt.Printf("addr send:%s\n", hashPanic)
	for k, v := range addr {
		fmt.Printf("%d\t%s\n", k, v)
	}
	if *delete {
		fmt.Printf("delete by hash:%s", hashPanic)
		err := cDB.DeleteElectionResultByHash(hashPanic)
		if err != nil {
			panic(err)
		}
	}

}
