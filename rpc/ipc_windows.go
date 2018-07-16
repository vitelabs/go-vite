// +build windows

package rpc

import (
	"context"
	"net"

	"gopkg.in/natefinch/npipe.v2"
)

func IpcListen(endpoint string) (net.Listener, error) {
	return npipe.Listen(endpoint)
}

func newIPCConnection(ctx context.Context, endpoint string) (net.Conn, error) {
	return npipe.Dial(endpoint)
}
