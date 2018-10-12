package node

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/utils/flock"
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
	stop            chan struct{}
	lock            sync.RWMutex
	instanceDirLock flock.Releaser // prevents concurrent use of instance directory
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
		stop:         make(chan struct{}),
	}, nil
}

func (node *Node) Start() error {
	node.lock.Lock()
	defer node.lock.Unlock()

	log.Info(fmt.Sprintf("Check dataDir is OK ? "))
	if err := node.openDataDir(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("DataDir is OK. "))

	//wallet start
	log.Info(fmt.Sprintf("Begin Start Wallet... "))

	if err := node.startWallet(); err != nil {
		log.Error(fmt.Sprintf("Node startWallet error: %v", err))
		return err
	}
	//p2p\vite start
	log.Info(fmt.Sprintf("Begin Start P2P And Vite... "))

	if err := node.startP2pAndVite(); err != nil {
		log.Error(fmt.Sprintf("Node startP2pAndVite error: %v", err))
		return err
	}
	//rpc start
	log.Info(fmt.Sprintf("Begin Start RPC... "))
	if err := node.startRPC(); err != nil {
		log.Error(fmt.Sprintf("Node startRPC error: %v", err))
		return err
	}

	return nil
}

func (node *Node) Stop() error {
	node.lock.Lock()
	defer node.lock.Unlock()
	// unblock n.Wait
	defer close(node.stop)

	//wallet
	log.Info(fmt.Sprintf("Begin Stop Wallet... "))
	if err := node.stopWallet(); err != nil {
		log.Error(fmt.Sprintf("Node stopWallet error: %v", err))
	}

	//p2p
	log.Info(fmt.Sprintf("Begin Stop P2P... "))
	if err := node.stopP2P(); err != nil {
		log.Error(fmt.Sprintf("Node stopP2P error: %v", err))
	}

	//vite
	log.Info(fmt.Sprintf("Begin Stop Vite... "))
	if err := node.stopVite(); err != nil {
		log.Error(fmt.Sprintf("Node stopVite error: %v", err))
	}

	//rpc
	log.Info(fmt.Sprintf("Begin Stop RPD... "))
	if err := node.stopRPC(); err != nil {
		log.Error(fmt.Sprintf("Node stopRPC error: %v", err))
	}

	// Release instance directory lock.
	log.Info(fmt.Sprintf("Begin relaeck dataDir lock... "))
	if node.instanceDirLock != nil {
		if err := node.instanceDirLock.Release(); err != nil {
			log.Error("Can't release dataDir lock...", "err", err)
		} else {
			log.Info("The file lock has been released...")
		}
		node.instanceDirLock = nil
	}

	return nil
}

func (node *Node) Wait() {
	node.lock.RLock()

	if node.p2pServer == nil {
		node.lock.RUnlock()
		return
	}
	node.lock.RUnlock()
	<-node.stop
}

func (node *Node) Config() *Config {
	return node.config
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
	node.p2pServer, err = p2p.New(node.p2pConfig)
	if err != nil {
		log.Error(fmt.Sprintf("P2P new error: %v", err))
		return err
	}

	//Initialize the vite server
	node.viteServer, err = vite.New(node.viteConfig, node.walletManager)
	if err != nil {
		log.Error(fmt.Sprintf("Vite new error: %v", err))
		return err
	}

	//Protocols setting, maybe should move into module.Start()
	node.p2pServer.Protocols = append(node.p2pServer.Protocols, node.viteServer.Net().Protocols()...)

	// Start vite
	if e := node.viteServer.Init(); e != nil {
		log.Error(fmt.Sprintf("ViteServer init error: %v", e))
		return err
	}

	if e := node.viteServer.Start(node.p2pServer); e != nil {
		log.Error(fmt.Sprintf("ViteServer start error: %v", e))
		return err
	}

	// Start p2p
	if e := node.p2pServer.Start(); e != nil {
		log.Error(fmt.Sprintf("P2PServer start error: %v", e))
		return err
	}
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

	// open data dir
	if err := os.MkdirAll(node.config.DataDir, 0700); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Open NodeServer.DataDir:%v", node.config.DataDir))

	//Lock the instance directory to prevent concurrent use by another instance as well as accidental use of the instance directory as a database.
	lockDir := filepath.Join(node.config.DataDir, "LOCK")
	log.Info(fmt.Sprintf("Try to Lock NodeServer.DataDir,lockDir:%v", lockDir))
	release, _, err := flock.New(lockDir)
	if err != nil {
		log.Error(fmt.Sprintf("Directory locked failed,lockDir:%v", lockDir))
		return convertFileLockError(err)
	}
	log.Info(fmt.Sprintf("Directory locked successfully,lockDir:%v", lockDir))
	node.instanceDirLock = release

	// open p2p data dir
	if err := os.MkdirAll(node.p2pConfig.DataDir, 0700); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Open NodeServer.p2pConfig.DataDir:%v", node.p2pConfig.DataDir))

	//open wallet data dir
	if err := os.MkdirAll(node.walletConfig.DataDir, 0700); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Open NodeServer.walletConfig.DataDir:%v", node.walletConfig.DataDir))

	return nil
}
