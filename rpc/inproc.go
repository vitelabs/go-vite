package rpc

import (
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net"
	"net/rpc"
)

func DialInProc(handler *rpc.Server) *rpc.Client {
	p1, p2 := net.Pipe()
	go handler.ServeCodec(jsonrpc2.NewServerCodec(p2, handler))
	return rpc.NewClientWithCodec(jsonrpc2.NewClientCodec(p1))
}
