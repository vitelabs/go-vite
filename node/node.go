package node

import (
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/vite"
	"net"
	netrpc "net/rpc"
	"os"
	"path/filepath"
	"sync"
)

// Node is a container that manages p2p、rpc、vite modules
type Node struct {
	config *config.Config

	p2pConfig p2p.Config // p2p config
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

	stop   chan struct{} // Channel to wait for termination notifications
	lock   sync.RWMutex
	Logger log15.Logger
}

// New creates a new P2P node
func New(conf *config.Config) (*Node, error) {

	// Copy config and resolve the datadir so future changes to the current working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absDataDir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absDataDir
	}

	//TODO something wo don't need now

	return &Node{
		config:    conf,
		p2pConfig: *conf.P2P,
		Logger:    log15.New("module", "gvite/main"),
	}, nil
}

func (node *Node) Start() error {

	node.lock.Lock()
	defer node.lock.Unlock()

	// Short circuit if the node's already running
	if node.p2pServer != nil {
		return ErrNodeRunning
	}

	if err := node.openDataDir(); err != nil {
		return err
	}

	//Initialize the p2p server
	p2pServer := p2p.New(p2p.Config{})

	vite, err := vite.New(cfg)

	p2pServer.Protocols = append(p2pServer.Protocols, vite.protocols()...)

	p2pServer.Start()

	vite.Start(p2pServer)

	rpc_vite.StartIpcRpcEndpoint()

	return nil
}

func (node *Node) Stop() error {
	return nil
}

func (node *Node) openDataDir() error {

	if node.config.DataDir == "" {
		return nil
	}

	if err := os.MkdirAll(node.config.DataDir, 0700); err != nil {
		return err
	}

	//Lock the instance directory to prevent concurrent use by another instance as well as accidental use of the instance directory as a database.
	//TODO miss file lock(flock)

	return nil
}
