package rpc

import (
	"github.com/powerman/rpc-codec/jsonrpc2"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"github.com/vitelabs/go-vite/rpc/service"
)

var (
	host = "127.0.0.1"
	port = "8080"
)

func StartHttpTestServer() {
	rpc.Register(&service.ExampleSvc{})
	rpc.Register(&service.AccountChain{})

	listener, err := net.Listen("tcp", ":"+port)

	if err != nil {
		log.Fatal("Server listen error: ", err.Error())
	}

	defer listener.Close()

	log.Println("Server listen on " + port)

	http.Serve(listener, jsonrpc2.HTTPHandler(nil))

}
