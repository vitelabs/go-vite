// Copyright 2018 The go-ethereum Authors
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
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	log "github.com/vitelabs/go-vite/v2/log15"
)

// StartHTTPEndpoint starts the HTTP RPC endpoint, configured with cors/vhosts/modules
func StartHTTPEndpoint(endpoint string, privateEndpoint string, apis []API, modules []string, cors []string, vhosts []string, timeouts HTTPTimeouts, exposeAll bool) (net.Listener, *Server, net.Listener, *Server, error) {
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := NewServer()
	privateHandler := NewServer()
	privateBind := false
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return nil, nil, nil, nil, err
			}
			log.Debug("HTTP registered", "namespace", api.Namespace)
		} else if !api.Public {
			if err := privateHandler.RegisterName(api.Namespace, api.Service); err != nil {
				return nil, nil, nil, nil, err
			}
			privateBind = true
			log.Debug("Private HTTP API registered", "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener        net.Listener
		privateListener net.Listener
		err             error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, nil, nil, err
	}

	go NewHTTPServer(cors, vhosts, timeouts, handler).Serve(listener)

	if privateBind && len(privateEndpoint) > 0 {
		if privateListener, err = net.Listen("tcp", privateEndpoint); err != nil {
			return nil, nil, nil, nil, err
		}
		go NewHTTPServer(cors, vhosts, timeouts, privateHandler).Serve(privateListener)
	}

	return listener, handler, privateListener, privateHandler, err
}

// StartWSEndpoint starts chain websocket endpoint
func StartWSEndpoint(endpoint string, apis []API, modules []string, wsOrigins []string, exposeAll bool) (net.Listener, *Server, error) {

	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := NewServer()
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return nil, nil, err
			}
			log.Debug("WebSocket registered", "service", api.Service, "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}

	go NewWSServer(wsOrigins, handler).Serve(listener)

	return listener, handler, err

}

// StartIPCEndpoint starts an IPC endpoint.
/*
func StartIPCEndpoint(ipcEndpoint string, apis []API) (net.Listener, *Server, error) {
	// Register all the APIs exposed by the services.
	handler := NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
		log.Debug("IPC registered", "namespace", api.Namespace)
	}
	// All APIs registered, start the IPC listener.
	listener, err := ipcListen(ipcEndpoint)

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		log.Info("receiver term sig")
		if listener != nil {
			listener.Close()
		}
	}()

	if err != nil {
		return nil, nil, err
	}

	go handler.ServeListener(listener)

	return listener, handler, nil
}
*/

// In this modified function:
// - The `networkType` parameter allows you to specify "mainnet" or "testnet."
// - The `uniqueEndpoint` is constructed by appending the network type to the original `ipcEndpoint`.
// - This ensures that the IPC endpoints are distinct for each network.

// StartIPCEndpoint starts an IPC endpoint for either mainnet or testnet.
func StartIPCEndpoint(networkType string, ipcEndpoint string, apis []API) (net.Listener, *Server, error) {
	// Construct a unique IPC endpoint by appending the network type.
	uniqueEndpoint := ipcEndpoint + "_" + networkType

	// Register APIs as before.
	handler := NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
		log.Debug("IPC registered", "namespace", api.Namespace)
	}

	// Start the IPC listener.
	listener, err := ipcListen(uniqueEndpoint)
	if err != nil {
		return nil, nil, err
	}

	// Handle termination signals.
	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		log.Info("receiver term sig")
		if listener != nil {
			listener.Close()
		}
	}()

	// Serve the listener.
	go handler.ServeListener(listener)

	return listener, handler, nil
}

// StartWSEndpoint starts chain websocket endpoint
func StartWSCliEndpoint(u *url.URL, apis []API, modules []string, exposeAll bool) (*WebSocketCli, *Server, error) {

	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := NewServer()
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return nil, nil, err
			}
			log.Debug("WebSocket registered", "service", api.Service, "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		ws  *WebSocketCli
		err error
	)

	ws = NewWSCli(u, handler)

	go ws.Handle()

	return ws, handler, err

}
