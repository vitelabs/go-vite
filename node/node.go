package node

import (
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/vite"
	"net"
	netrpc "net/rpc"
)

// Node is a container that manages p2p、rpc、vite modules
type Node struct {
	config *config.Config

	p2pConfig *config.P2P // p2p config
	p2pServer *p2p.Server

	vite *vite.Vite

	rpcAPIs []rpc.API // List of APIs currently provided by the node

	ipcEndpoint string         // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener   // IPC RPC listener socket to serve API requests
	ipcHandler  *netrpc.Server // IPC RPC request handler to process the API requests

	httpEndpoint  string         // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string       // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener   // HTTP RPC listener socket to server API requests
	httpHandler   *netrpc.Server // HTTP RPC request handler to process the API requests

	wsEndpoint string         // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener   // Websocket RPC listener socket to server API requests
	wsHandler  *netrpc.Server // Websocket RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications

	Logger log15.Logger
}

func New(conf *config.Config) (*Node, error) {

	return nil, nil
}

func (node *Node) Start() error {

	//TODO miss Lock

	if node.p2pServer != nil {
		return ErrNodeRunning
	}

	//TODO miss dataDir judge

	node.p2pConfig = node.config.P2P

	return nil
}

func (node *Node) Stop() error {
	return nil
}
