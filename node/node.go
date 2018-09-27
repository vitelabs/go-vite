package node

import (
	"fmt"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Node is a container that manages p2p、rpc、vite modules
type Node struct {
	config *Config

	//wallet
	acctManager *wallet.Manager

	//vite
	viteConfig config.Config
	viteServer *vite.Vite

	//p2p
	p2pConfig p2p.Config
	p2pServer *p2p.Server

	//TODO: in the future need
	//rpcAPIs []rpc.API // List of APIs currently provided by the node

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener // HTTP RPC listener socket to server API requests
	httpHandler   *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server API requests
	wsHandler  *rpc.Server  // Websocket RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex
	log  log15.Logger
}

// New creates a new P2P node
func New(conf *Config) (*Node, error) {

	// Copy config and resolve the dataDir so future changes to the current working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absDataDir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		//TODO
		conf.DataDir = absDataDir
	}
	conf.P2P.Datadir = conf.DataDir

	return &Node{
		config:       conf,
		acctManager:  wallet.New(&wallet.Config{DataDir: conf.DataDir}),
		p2pConfig:    conf.makeP2PConfig(),
		viteConfig:   conf.makeViteConfig(),
		ipcEndpoint:  conf.IPCEndpoint(),
		httpEndpoint: conf.HTTPEndpoint(),
		wsEndpoint:   conf.WSEndpoint(),
		log:          log15.New("module", "gvite/node"),
	}, nil
}

func (node *Node) Start() error {

	node.lock.Lock()
	defer node.lock.Unlock()

	//wallet start
	node.acctManager.Start()

	// Short circuit if the node's already running
	if node.p2pServer != nil {
		return ErrNodeRunning
	}

	if err := node.openDataDir(); err != nil {
		return err
	}

	//Initialize the p2p server
	node.p2pServer = p2p.New(node.p2pConfig)

	//Initialize the vite server
	vite, err := vite.New(&node.viteConfig)
	if err != nil {
		node.log.Error(fmt.Sprint("vite new error: %v", err))
		return err
	}
	node.viteServer = vite

	node.p2pServer.Protocols = append(node.p2pServer.Protocols, vite.Protocols()...)

	node.p2pServer.Start()

	// Start vite
	node.viteServer.Start(node.p2pServer)

	// Start rpc
	// Get all the possible Apis
	apis := rpcapi.GetAllApis(node.viteServer)

	if node.config.IPCEnabled {
		if err := node.startIPC(apis); err != nil {
			node.stopIPC()
			return err
		}
	}

	if node.config.RPCEnabled {
		if err := node.startHTTP(node.httpEndpoint, apis, nil, nil, nil, rpc.HTTPTimeouts{}); err != nil {
			node.stopIPC()
			return err
		}
	}

	if node.config.WSEnabled {
		if err := node.startWS(node.wsEndpoint, apis, nil, nil, true); err != nil {
			node.stopHTTP()
			node.stopIPC()
			return err
		}
	}

	return nil
}

func (node *Node) Stop() error {

	node.lock.Lock()
	defer node.lock.Unlock()

	// Short circuit if the node's not running
	if node.p2pServer == nil {
		return ErrNodeStopped
	}

	//wallet
	node.acctManager.Stop()

	//p2p
	node.p2pServer.Stop()

	//vite
	node.viteServer.Stop()

	//rpc
	node.stopWS()
	node.stopHTTP()
	node.stopIPC()

	// unblock n.Wait
	close(node.stop)

	return nil
}

func (node *Node) Wait() {
	node.lock.RLock()
	if node.p2pServer == nil {
		node.lock.RUnlock()
	}

	stop := node.stop
	node.lock.RUnlock()

	<-stop
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

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC(apis []rpc.API) error {
	if n.ipcEndpoint == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(n.ipcEndpoint, apis)
	if err != nil {
		return err
	}
	n.ipcListener = listener
	n.ipcHandler = handler
	n.log.Info("IPC endpoint opened", "url", n.ipcEndpoint)
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil

		n.log.Info("IPC endpoint closed", "endpoint", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := rpc.StartHTTPEndpoint(endpoint, apis, modules, cors, vhosts, timeouts)
	if err != nil {
		return err
	}
	n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))
	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		n.log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, apis []rpc.API, modules []string, wsOrigins []string, exposeAll bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := rpc.StartWSEndpoint(endpoint, apis, modules, wsOrigins, exposeAll)
	if err != nil {
		return err
	}
	n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))
	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		n.log.Info("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", n.wsEndpoint))
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}
