package main

import "flag"

func main() {
	var isInit= false
	flag.BoolVar(&isInit, "init", false, "Init ledger")

	flag.Parse()
	if isInit {
		writeGenesisBlocks()
	}
	//test()
	//testCompare()
	//testWrite()
}

//func testWrite()  {
//	iterKey := []byte("c.%01.%001")
//	db, _ := vitedb.GetLDBDataBase(vitedb.DB_BLOCK)
//	db.Leveldb.Put(iterKey, []byte("hello world"), nil)
//
//	iterKey2 := []byte("c.%01.%002")
//	db.Leveldb.Put(iterKey2, []byte("hello world2"), nil)
//}
//func test () {
//	iterKey := []byte("c.%01.%01.")
//	db, _ := vitedb.GetLDBDataBase(vitedb.DB_BLOCK)
//	//value, err := db.Leveldb.Get(iterKey, nil)
//	//fmt.Println(err)
//	//fmt.Println(string(value))
//	fmt.Println(string(util.BytesPrefix(iterKey).Start))
//	fmt.Println(string(util.BytesPrefix(iterKey).Limit))
//	//fmt.Println()
//	iter := db.Leveldb.NewIterator(util.BytesPrefix(iterKey), nil)
//	//
//	for iter.Next() {
//		fmt.Println(string(iter.Key()))
//		fmt.Println(iter.Value())
//	}
//}


