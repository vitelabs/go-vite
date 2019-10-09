package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/consensus/cdb"
)

var index = flag.Uint64("index", 0, "index")
var dir = flag.String("dataDir", "", "consensus data dir")

func main() {
	flag.Parse()
	if *index == 0 {
		panic("error index")
	}
	if *dir == "" {
		panic("err dir")
	}
	d, err := leveldb.OpenFile(*dir, nil)
	if err != nil {
		panic(err)
	}

	db := cdb.NewConsensusDB(d)
	point, err := db.GetPointByHeight(cdb.IndexPointDay, *index)
	if err != nil {
		panic(err)
	}
	bytes, err := json.Marshal(point)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bytes))
	err = db.DeletePointByHeight(cdb.IndexPointDay, *index)
	if err != nil {
		panic(err)
	}
}
