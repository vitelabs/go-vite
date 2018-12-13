package node

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/pkg/errors"

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
	p2pServer p2p.Server

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

	wsCli *rpc.WebSocketCli

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

	//wallet start
	log.Info(fmt.Sprintf("Begin Start Wallet... "))
	if err := node.startWallet(); err != nil {
		log.Error(fmt.Sprintf("startWallet error: %v", err))
		return err
	}

	// should after startWallet
	if node.config.MinerEnabled && node.config.CoinBase != "" {
		if addr, err := node.extractAddr(); err != nil {
			log.Error(fmt.Sprintf("extract address error: %v", err))
		} else {
			node.p2pConfig.ExtNodeData = addr[:]
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
	node.p2pServer.Config().Protocols = append(node.p2pServer.Config().Protocols, node.viteServer.Net().Protocols()...)

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
	if len(node.config.DashboardTargetURL) > 0 {
		apis := rpcapi.GetPublicApis(node.viteServer)
		if len(node.config.PublicModules) != 0 {
			apis = rpcapi.GetApis(node.viteServer, node.config.PublicModules...)
		}

		targetUrl := node.config.DashboardTargetURL + "/ws/gvite/" + strconv.FormatUint(uint64(node.config.NetID), 10) + "@" + hex.EncodeToString(node.p2pServer.Config().PrivateKey.PubByte())

		u, e := url.Parse(targetUrl)
		if e != nil {
			return e
		}
		if u.Scheme != "ws" && u.Scheme != "wss" {
			return errors.New("DashboardTargetURL need match WebSocket Protocol.")
		}

		cli, server, e := rpc.StartWSCliEndpoint(u, apis, nil, node.config.WSExposeAll)
		if e != nil {
			cli.Close()
			server.Stop()
			return e
		} else {
			node.wsCli = cli
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

func (node *Node) extractAddr() (addr *types.Address, err error) {
	coinBase := node.config.CoinBase

	var index uint32
	addr, index, err = parseCoinBase(coinBase)

	if err != nil {
		log.Error(fmt.Sprintf("coinBase parse fail. %v", coinBase), "err", err)
		return
	}
	err = node.walletManager.MatchAddress(node.config.EntropyStorePath, *addr, index)

	if err != nil {
		log.Error(fmt.Sprintf("coinBase is not child of entropyStore, coinBase is : %v", coinBase), "err", err)
	}

	return
}

func parseCoinBase(coinBase string) (*types.Address, uint32, error) {
	splits := strings.Split(coinBase, ":")
	if len(splits) != 2 {
		return nil, 0, errors.New("len is not equals 2.")
	}
	i, err := strconv.Atoi(splits[0])
	if err != nil {
		return nil, 0, err
	}
	addr, err := types.HexToAddress(splits[1])
	if err != nil {
		return nil, 0, err
	}

	return &addr, uint32(i), nil
}
