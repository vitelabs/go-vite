package node

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vitelabs/go-vite/cmd/utils/flock"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/pow/remote"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
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

func (node *Node) Prepare() error {
	node.lock.Lock()
	defer node.lock.Unlock()

	log.Info(fmt.Sprintf("Check dataDir is OK ? "))
	if err := node.openDataDir(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("DataDir is OK. "))

	//prepare node
	log.Info(fmt.Sprintf("Begin Prepare node... "))
	//prepare wallet
	if node.walletConfig == nil {
		return ErrWalletConfigNil
	}

	if node.walletManager != nil {
		return ErrNodeRunning
	}
	node.walletManager = wallet.New(node.walletConfig)

	//wallet start
	log.Info(fmt.Sprintf("Begin Start Wallet... "))
	if err := node.startWallet(); err != nil {
		log.Error(fmt.Sprintf("startWallet error: %v", err))
		return err
	}

	//prepare p2p
	if node.p2pConfig == nil {
		return ErrP2PConfigNil
	}

	if node.p2pServer != nil {
		return ErrNodeRunning
	}

	if node.viteServer != nil {
		return ErrNodeRunning
	}

	// extract p2p privateKey from node.walletManager
	if node.p2pConfig.PrivateKey == nil {
		if priv, err := node.extractPrivateKeyFromCoinbase(); err == nil {
			node.p2pConfig.PrivateKey = priv
		}
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

	//init rpc_PowServerUrl
	remote.InitRawUrl(node.Config().PowServerUrl)
	pow.Init(node.Config().VMTestParamEnabled)

	// Start vite
	if err := node.viteServer.Init(); err != nil {
		log.Error(fmt.Sprintf("ViteServer init error: %v", err))
		return err
	}
	return nil
}

func (node *Node) Start() error {
	node.lock.Lock()
	defer node.lock.Unlock()

	//p2p\vite start
	log.Info(fmt.Sprintf("Begin Start Vite... "))
	if err := node.startVite(); err != nil {
		log.Error(fmt.Sprintf("ViteServer start error: %v", err))
		return err
	}

	// Start p2p
	log.Info(fmt.Sprintf("Begin Start P2p... "))
	if err := node.p2pServer.Start(); err != nil {
		log.Error(fmt.Sprintf("P2PServer start error: %v", err))
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

func (node *Node) Vite() *vite.Vite {
	return node.viteServer
}

func (node *Node) Config() *Config {
	return node.config
}

func (node *Node) ViteServer() *vite.Vite {
	return node.viteServer
}

func (node *Node) WalletManager() *wallet.Manager {
	return node.walletManager
}

//wallet start
func (node *Node) startWallet() error {
	node.walletManager.Start()
	//unlock account
	if node.config.EntropyStorePath != "" {

		if err := node.walletManager.AddEntropyStore(node.config.EntropyStorePath); err != nil {
			log.Error(fmt.Sprintf("node.walletManager.AddEntropyStore error: %v", err))
			return err
		}

		entropyStoreManager, err := node.walletManager.GetEntropyStoreManager(node.config.EntropyStorePath)

		if err != nil {
			log.Error(fmt.Sprintf("node.walletManager.GetEntropyStoreManager error: %v", err))
			return err
		}

		//unlock
		if err := entropyStoreManager.Unlock(node.config.EntropyStorePassword); err != nil {
			log.Error(fmt.Sprintf("entropyStoreManager.Unlock error: %v", err))
			return err
		}

	}

	return nil
}

func (node *Node) startVite() error {
	return node.viteServer.Start(node.p2pServer)
}

func (node *Node) startRPC() error {

	// Init rpc log
	rpcapi.Init(node.config.DataDir, node.config.LogLevel, node.config.TestTokenHexPrivKey, node.config.TestTokenTti)

	// Start the various API endpoints, terminating all in case of errors
	if err := node.startInProcess(node.GetInProcessApis()); err != nil {
		return err
	}

	// Start rpc
	if node.config.IPCEnabled {
		if err := node.startIPC(node.GetIpcApis()); err != nil {
			node.stopInProcess()
			return err
		}
	}

	if node.config.RPCEnabled {
		apis := rpcapi.GetPublicApis(node.viteServer)
		if len(node.config.PublicModules) != 0 {
			apis = rpcapi.GetApis(node.viteServer, node.config.PublicModules...)
		}
		if err := node.startHTTP(node.httpEndpoint, apis, nil, node.config.HTTPCors, node.config.HttpVirtualHosts, rpc.HTTPTimeouts{}, node.config.HttpExposeAll); err != nil {
			node.stopInProcess()
			node.stopIPC()
			return err
		}
	}

	if node.config.WSEnabled {
		apis := rpcapi.GetPublicApis(node.viteServer)
		if len(node.config.PublicModules) != 0 {
			apis = rpcapi.GetApis(node.viteServer, node.config.PublicModules...)
		}
		if err := node.startWS(node.wsEndpoint, apis, nil, node.config.WSOrigins, node.config.WSExposeAll); err != nil {
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

func (node *Node) extractPrivateKeyFromCoinbase() (priv ed25519.PrivateKey, err error) {
	addr, index, err := parseCoinbase(node.config.CoinBase)
	if err != nil {
		return
	}

	err = node.walletManager.MatchAddress(node.config.EntropyStorePath, addr, uint32(index))
	if err != nil {
		return
	}

	var em *entropystore.Manager
	em, err = node.walletManager.GetEntropyStoreManager(node.config.EntropyStorePath)
	if err != nil {
		return
	}

	_, key, err := em.DeriveForIndexPath(uint32(index))
	if err != nil {
		return
	}

	return key.PrivateKey()
}

func parseCoinbase(coinbase string) (addr types.Address, index int, err error) {
	splits := strings.Split(coinbase, ":")
	if len(splits) != 2 {
		err = errors.New("len is not equals 2")
		return
	}

	index, err = strconv.Atoi(splits[0])
	if err != nil {
		return
	}

	addr, err = types.HexToAddress(splits[1])
	if err != nil {
		return
	}

	return
}
