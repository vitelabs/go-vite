// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received chain copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"fmt"
	"net"

	log "github.com/vitelabs/go-vite/log15"
)

// ServeListener accepts connections on l, serving JSON-RPC on them.
func (srv *Server) ServeListener(l net.Listener) error {
	fmt.Println("Vite rpc start success!")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error("ServeListener ", "err", err)
			return err
		}
		//if netutil.IsTemporaryError(err) {
		//	log.Warn("RPC accept error", "err", err)
		//	continue
		//} else if err != nil {
		//	return err
		//}
		if conn != nil {
			log.Info("Accepted connection", conn.RemoteAddr(), "addr", conn.LocalAddr())
		}
		go srv.ServeCodec(NewJSONCodec(conn), OptionMethodInvocation|OptionSubscriptions)
	}
}

// DialIPC create chain new IPC client that connects to the given endpoint. On Unix it assumes
// the endpoint is the full path to chain unix socket, and Windows the endpoint is an
// identifier for chain named pipe.
//
// The context is used for the initial connection establishment. It does not
// affect subsequent interactions with the client.
func DialIPC(ctx context.Context, endpoint string) (*Client, error) {
	return newClient(ctx, func(ctx context.Context) (net.Conn, error) {
		return newIPCConnection(ctx, endpoint)
	})
}
