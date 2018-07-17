package rpc

import (
	"context"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/vitelabs/go-vite/log"
	"net"
	"net/rpc"
	"runtime"
)

func ServeListener(srv *rpc.Server, l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Warn("RPC accept error", "err", err)
			return err
		}
		log.Trace("Accepted connection", "addr", conn.RemoteAddr())
		go srv.ServeCodec(jsonrpc2.NewServerCodec(conn, srv))
	}
}

func DialIPC(ctx context.Context, endpoint string) (*rpc.Client, error) {
	conn, err := newIPCConnection(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return rpc.NewClientWithCodec(jsonrpc2.NewClientCodec(conn)), nil

}

func DefaultIpcFile() string {
	endpoint := "vite.ipc"
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\vite.ipc`
	}
	return endpoint
}