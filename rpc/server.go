package rpc

import (
	"log"
	"net"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net/rpc"
	"net/http"
	"go-vite/rpc/service"
)

var (
	host = "127.0.0.1"
	port  = "8080"
)

func StartServer () {
	rpc.Register(&service.ExampleSvc{})
	rpc.Register(&service.AccountChain{})

	listener, err := net.Listen("tcp", ":" + port)

	if err != nil {
		log.Fatal("Server listen error: ", err.Error())
	}

	defer listener.Close()

	log.Println("Server listen on " + port)

	http.Serve(listener, jsonrpc2.HTTPHandler(nil))

}