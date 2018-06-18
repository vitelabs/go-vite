package main

import (
	"go-vite/rpc"
)

func main()  {
	// http://localhost:8080/  {"jsonrpc": "2.0", "method": "ExampleSvc.Sum", "params": [4, 5], "id": 1}
	rpc.StartServer()
}