package node

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/cmd/utils/flock"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/metrics"
	"github.com/vitelabs/go-vite/metrics/influxdb"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/pow/remote"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/rpcapi/api/filters"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/entropystore"
)

var (
	log = log15.New("module", "gvite/node")
)

// Node is chain container that manages p2p、rpc、vite modules
type Node struct {
	config *Config

	//wallet
	walletConfig  *wallet.Config
	walletManager *wallet.Manager

	//vite
	viteConfig *config.Config
	viteServer *vite.Vite

	// metrics
	metricsConfig *metrics.Config
	ifxReporter   *influxdb.Reporter

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
		config:        conf,
		walletConfig:  conf.makeWalletConfig(),
		viteConfig:    conf.makeViteConfig(),
		metricsConfig: conf.makeMetricsConfig(),
		ipcEndpoint:   conf.IPCEndpoint(),
		httpEndpoint:  conf.HTTPEndpoint(),
		wsEndpoint:    conf.WSEndpoint(),
		stop:          make(chan struct{}),
	}, nil
}

func (node *Node) Prepare() (err error) {
	node.lock.Lock()
	defer node.lock.Unlock()

	log.Info(fmt.Sprintf("Check dataDir is OK ? "))
	if err = node.openDataDir(); err != nil {
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

	if node.viteServer != nil {
		return ErrNodeRunning
	}

	//wallet start
	log.Info(fmt.Sprintf("Begin Start Wallet... "))
	if err = node.startWallet(); err != nil {
		log.Error(fmt.Sprintf("startWallet error: %v", err))
		return err
	}

	//Initialize the vite server
	node.viteServer, err = vite.New(node.viteConfig, node.walletManager)
	if err != nil {
		log.Error(fmt.Sprintf("Vite new error: %v", err))
		return err
	}

	//init rpc_PowServerUrl
	remote.InitRawUrl(node.Config().PowServerUrl)
	pow.Init(node.Config().VMTestParamEnabled)

	// Start vite
	if err = node.viteServer.Init(); err != nil {
		log.Error(fmt.Sprintf("ViteServer init error: %v", err))
		return err
	}
	return nil
}

func (node *Node) Start() error {
	node.lock.Lock()
	defer node.lock.Unlock()

	// metrics start
	node.startMetrics()

	//p2p\vite start
	log.Info(fmt.Sprintf("Begin Start Vite... "))
	if err := node.startVite(); err != nil {
		log.Error(fmt.Sprintf("ViteServer start error: %v", err))
		return err
	}

	//rpc start
	log.Info(fmt.Sprintf("Begin Start RPC... "))
	if err := node.startRPC(); err != nil {
		log.Error(fmt.Sprintf("Node startRPC error: %v", err))
		return err
	}
	monitor.InitNTPChecker(log)

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

	//vite
	log.Info(fmt.Sprintf("Begin Stop Vite... "))
	if err := node.stopVite(); err != nil {
		log.Error(fmt.Sprintf("Node stopVite error: %v", err))
	}

	// metrics influxdb reporter
	log.Info(fmt.Sprintf("Begin Stop Metrics... "))
	node.stopMetrics()

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
	<-node.stop
}

func (node *Node) Vite() *vite.Vite {
	return node.viteServer
}

func (node *Node) Config() *Config {
	return node.config
}

func (node *Node) ViteConfig() *config.Config {
	return node.viteConfig
}

func (node *Node) ViteServer() *vite.Vite {
	return node.viteServer
}

func (node *Node) WalletManager() *wallet.Manager {
	return node.walletManager
}

//wallet start
func (node *Node) startWallet() (err error) {
	err = node.walletManager.Start()
	if err != nil {
		return
	}
	//unlock account
	if node.config.EntropyStorePath != "" {

		if err = node.walletManager.AddEntropyStore(node.config.EntropyStorePath); err != nil {
			log.Error(fmt.Sprintf("node.walletManager.AddEntropyStore error: %v", err))
			return err
		}

		var entropyStoreManager *entropystore.Manager
		entropyStoreManager, err = node.walletManager.GetEntropyStoreManager(node.config.EntropyStorePath)

		if err != nil {
			log.Error(fmt.Sprintf("node.walletManager.GetEntropyStoreManager error: %v", err))
			return err
		}

		//unlock
		if err = entropyStoreManager.Unlock(node.config.EntropyStorePassword); err != nil {
			log.Error(fmt.Sprintf("entropyStoreManager.Unlock error: %v", err))
			return err
		}

	}

	return nil
}
func (node *Node) startMetrics() {
	// init metrics args
	metricsCfg := node.metricsConfig
	if metricsCfg == nil {
		return
	}
	if metricsCfg.IsInfluxDBEnable == false || metricsCfg.InfluxDBInfo == nil {
		log.Info("influxdb export disable or influxdbinfo of reporter is not complete")
		metricsCfg.IsInfluxDBEnable = false
	}

	metrics.InitMetrics(metricsCfg.IsEnable, metricsCfg.IsInfluxDBEnable)

	if metrics.MetricsEnabled {
		log.Info("start metrics collection")
		go metrics.CollectProcessMetrics(3 * time.Second)

		if metrics.InfluxDBExportEnable {
			influxDBInfo := metricsCfg.InfluxDBInfo

			rp, err := influxdb.NewReporter(metrics.DefaultRegistry, 10*time.Second,
				influxDBInfo.Endpoint, influxDBInfo.Database, influxDBInfo.Username, influxDBInfo.Password,
				"monitor", map[string]string{"host": influxDBInfo.HostTag})
			if err != nil || rp == nil {
				log.Error(fmt.Sprintf("new influxdb reporter err: %v", err))
				return
			}
			node.ifxReporter = rp
			log.Info("start influxdb export")
			node.ifxReporter.Start()
		}
	}
}

func (node *Node) stopMetrics() {
	if node.ifxReporter != nil {
		log.Info("stop influxdb export")
		node.ifxReporter.Stop()
	}
}

func (node *Node) startVite() error {
	return node.viteServer.Start()
}

func (node *Node) startRPC() error {

	// Init rpc log
	rpcapi.Init(node.config.DataDir, node.config.LogLevel, node.config.TestTokenHexPrivKey, node.config.TestTokenTti, uint(node.config.NetID), node.config.TxDexEnable)

	// Start the various API endpoints, terminating all in case of errors
	if err := node.startInProcess(node.GetInProcessApis()); err != nil {
		return err
	}

	// start event system
	if node.config.SubscribeEnabled {
		filters.Es = filters.NewEventSystem(node.Vite())
		filters.Es.Start()
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

		targetUrl := node.config.DashboardTargetURL + "/ws/gvite/" + strconv.FormatUint(uint64(node.config.NetID), 10) + "@" + node.Vite().Net().Info().ID.String()

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
	if filters.Es != nil {
		filters.Es.Stop()
	}
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

	//Lock the instance directory to prevent concurrent use by another instance as well as accidental use of the instance directory as chain database.
	lockDir := filepath.Join(node.config.DataDir, "LOCK")
	log.Info(fmt.Sprintf("Try to Lock NodeServer.DataDir,lockDir:%v", lockDir))
	release, _, err := flock.New(lockDir)
	if err != nil {
		log.Error(fmt.Sprintf("Directory locked failed,lockDir:%v", lockDir))
		return convertFileLockError(err)
	}
	log.Info(fmt.Sprintf("Directory locked successfully,lockDir:%v", lockDir))
	node.instanceDirLock = release

	//open wallet data dir
	if err := os.MkdirAll(node.walletConfig.DataDir, 0700); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Open NodeServer.walletConfig.DataDir:%v", node.walletConfig.DataDir))

	return nil
}

func parseCoinBase(coinBase string) (types.Address, uint32, error) {
	splits := strings.Split(coinBase, ":")
	if len(splits) != 2 {
		return types.Address{}, 0, errors.New("len is not equals 2.")
	}
	i, err := strconv.Atoi(splits[0])
	if err != nil {
		return types.Address{}, 0, err
	}
	addr, err := types.HexToAddress(splits[1])
	if err != nil {
		return types.Address{}, 0, err
	}

	return addr, uint32(i), nil
}
