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
	"sync"
)

var (
	log = log15.New("module", "gvite/node")
)

// Node is a container that manages p2p、rpc、vite modules
type Node struct {
	config *Config

	//wallet
	walletConfig  *wallet.Config
	walletManager *wallet.Manager

	//p2p
	p2pConfig *p2p.Config
	p2pServer *p2p.Server

	//vite
	viteConfig *config.Config
	viteServer *vite.Vite

	// List of APIs currently provided by the node
	rpcAPIs          []rpc.API
	inProcessHandler *rpc.Server

	ipcEndpoint string
	ipcListener net.Listener
	ipcHandler  *rpc.Server

	httpEndpoint  string
	httpWhitelist []string
	httpListener  net.Listener
	httpHandler   *rpc.Server

	wsEndpoint string
	wsListener net.Listener
	wsHandler  *rpc.Server

	// Channel to wait for termination notifications
	stop chan struct{}
	lock sync.RWMutex
}

func New(conf *Config) (*Node, error) {
	return &Node{
		config:       conf,
		walletConfig: conf.makeWalletConfig(),
		p2pConfig:    conf.makeP2PConfig(),
		viteConfig:   conf.makeViteConfig(),
		ipcEndpoint:  conf.IPCEndpoint(),
		httpEndpoint: conf.HTTPEndpoint(),
		wsEndpoint:   conf.WSEndpoint(),
	}, nil
}

func (node *Node) Start() error {
	node.lock.Lock()
	defer node.lock.Unlock()

	if err := node.openDataDir(); err != nil {
		return err
	}

	//wallet start
	node.startWallet()
	//p2p\vite start
	node.startP2pAndVite()
	//rpc start
	node.startRPC()

	return nil
}

func (node *Node) Stop() error {
	node.lock.Lock()
	defer node.lock.Unlock()
	// unblock n.Wait
	defer close(node.stop)

	//wallet
	node.stopWallet()

	//p2p
	node.stopP2P()

	//vite
	node.stopVite()

	//rpc
	node.stopRPC()

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

//wallet start
func (node *Node) startWallet() error {

	if node.walletConfig == nil {
		return ErrWalletConfigNil
	}

	if node.walletManager != nil {
		return ErrNodeRunning
	}

	node.walletManager = wallet.New(node.walletConfig)
	node.walletManager.Start()

	return nil
}

//p2p、vite safe start
func (node *Node) startP2pAndVite() error {

	if node.p2pConfig == nil {
		return ErrP2PConfigNil
	}

	if node.p2pServer != nil {
		return ErrNodeRunning
	}

	if node.viteServer != nil {
		return ErrNodeRunning
	}

	var err error
	node.p2pServer, err = p2p.New(*node.p2pConfig)
	if err != nil {
		log.Error(fmt.Sprintf("p2p new error: %v", err))
		return err
	}

	//Initialize the vite server
	node.viteServer, err = vite.New(node.viteConfig)
	if err != nil {
		log.Error(fmt.Sprintf("vite new error: %v", err))
		return err
	}

	node.p2pServer.Protocols = append(node.p2pServer.Protocols, node.viteServer.Protocols()...)

	// Start p2p
	if e := node.p2pServer.Start(); e != nil {
		log.Error(fmt.Sprintf("p2pServer start error: %v", e))
		return err
	}

	// Start vite
	node.viteServer.Start(node.p2pServer)

	return nil
}

func (node *Node) startRPC() error {

	// Get all the possible APIs
	node.rpcAPIs = rpcapi.GetAllApis(node.viteServer)

	// Start the various API endpoints, terminating all in case of errors
	if err := node.startInProcess(node.rpcAPIs); err != nil {
		return err
	}

	// Start rpc
	if node.config.IPCEnabled {
		if err := node.startIPC(node.rpcAPIs); err != nil {
			node.stopInProcess()
			return err
		}
	}

	if node.config.RPCEnabled {
		if err := node.startHTTP(node.httpEndpoint, node.rpcAPIs, nil, nil, nil, rpc.HTTPTimeouts{}); err != nil {
			node.stopInProcess()
			node.stopIPC()
			return err
		}
	}

	if node.config.WSEnabled {
		if err := node.startWS(node.wsEndpoint, node.rpcAPIs, nil, nil, true); err != nil {
			node.stopInProcess()
			node.stopIPC()
			node.stopHTTP()
			return err
		}
	}

	return nil
}

func (node *Node) stopWallet() error {

	if node.walletManager == nil {
		return ErrNodeStopped
	}

	node.walletManager.Stop()
	return nil
}

func (node *Node) stopP2P() error {

	if node.p2pServer == nil {
		return ErrNodeStopped
	}

	node.p2pServer.Stop()
	return nil
}

func (node *Node) stopVite() error {

	if node.viteServer == nil {
		return ErrNodeStopped
	}

	node.viteServer.Stop()
	return nil
}

func (node *Node) stopRPC() error {
	node.stopWS()
	node.stopHTTP()
	node.stopIPC()
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
