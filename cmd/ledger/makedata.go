package main

import (
	"flag"
)

func main()  {
	var isInit = false
	flag.BoolVar(&isInit, "init", false, "Init ledger")

	flag.Parse()
	if isInit {
		writeGenesisBlocks()
	}
}


