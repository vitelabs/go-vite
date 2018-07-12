package main

import "github.com/vitelabs/go-vite/rpc"

func main() {

	client := rpc.JsonrpcClient("http://127.0.0.1:8080")
	var i int
	err := client.Call("ExampleSvc.Sum", &[2]int{111, 2}, &i)
	if err != nil {
		panic(err)
	}

	print(i)
}
