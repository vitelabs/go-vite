package nodemanager

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"path/filepath"
)

var defaultNodeConfigFileName = "node_config.json"

type FullNodeMaker struct {
}

func (maker FullNodeMaker) MakeNode(ctx *cli.Context) *node.Node {

	// 1: Make Node.Config
	nodeConfig := maker.MakeNodeConfig(ctx)
	log.Info(fmt.Sprintf("NodeConfig info: %v", nodeConfig))
	// 2: New Node
	node, err := node.New(nodeConfig)

	if err != nil {
		log.Error("Failed to create the node: %v", err)
	}
	return node
}

func (maker FullNodeMaker) MakeNodeConfig(ctx *cli.Context) *node.Config {

	cfg := node.DefaultNodeConfig
	log.Info(fmt.Sprintf("DefaultNodeconfig: %v", cfg))

	// 1: Load config file.
	loadNodeConfigFromFile(ctx, &cfg)
	log.Info(fmt.Sprintf("After load config file: %v", cfg))

	// 2: Apply flags, Overwrite the configuration file configuration
	mappingNodeConfig(ctx, &cfg)
	log.Info(fmt.Sprintf("After mapping cmd input: %v", cfg))

	// 3: Override any default configs for hard coded networks.
	overrideNodeConfigs(ctx, &cfg)
	log.Info(fmt.Sprintf("Last override config: %v", cfg))

	// 4: Config log to file
	makeRunLogFile(&cfg)

	return &cfg
}

// SetNodeConfig applies node-related command line flags to the config.
func mappingNodeConfig(ctx *cli.Context, cfg *node.Config) {

	//Global Config
	if dataDir := ctx.GlobalString(utils.DataDirFlag.Name); len(dataDir) > 0 {
		cfg.DataDir = dataDir
	}

	//Wallet
	if keyStoreDir := ctx.GlobalString(utils.KeyStoreDirFlag.Name); len(keyStoreDir) > 0 {
		cfg.KeyStoreDir = keyStoreDir
	}

	//Network Config
	if identity := ctx.GlobalString(utils.IdentityFlag.Name); len(identity) > 0 {
		cfg.Identity = identity
	}

	if ctx.GlobalIsSet(utils.MaxPeersFlag.Name) {
		cfg.MaxPeers = ctx.GlobalUint(utils.MaxPeersFlag.Name)
	}

	if ctx.GlobalIsSet(utils.MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = ctx.GlobalUint(utils.MaxPendingPeersFlag.Name)
	}

	if ctx.GlobalIsSet(utils.ListenPortFlag.Name) {
		cfg.Port = ctx.GlobalUint(utils.ListenPortFlag.Name)
	}

	if nodeKeyHex := ctx.GlobalString(utils.NodeKeyHexFlag.Name); len(nodeKeyHex) > 0 {
		cfg.SetPrivateKey(nodeKeyHex)
	}

	//Ipc Config
	if ctx.GlobalIsSet(utils.IPCEnabledFlag.Name) {
		cfg.IPCEnabled = ctx.GlobalBool(utils.IPCEnabledFlag.Name)
	}

	if ipcPath := ctx.GlobalString(utils.IPCPathFlag.Name); len(ipcPath) > 0 {
		cfg.IPCPath = ipcPath
	}

	//Http Config
	if ctx.GlobalIsSet(utils.RPCEnabledFlag.Name) {
		cfg.RPCEnabled = ctx.GlobalBool(utils.RPCEnabledFlag.Name)
	}

	if httpHost := ctx.GlobalString(utils.RPCListenAddrFlag.Name); len(httpHost) > 0 {
		cfg.HttpHost = httpHost
	}

	if ctx.GlobalIsSet(utils.RPCPortFlag.Name) {
		cfg.HttpPort = ctx.GlobalInt(utils.RPCPortFlag.Name)
	}

	//WS Config
	if ctx.GlobalIsSet(utils.WSEnabledFlag.Name) {
		cfg.WSEnabled = ctx.GlobalBool(utils.WSEnabledFlag.Name)
	}

	if wsListenAddr := ctx.GlobalString(utils.WSListenAddrFlag.Name); len(wsListenAddr) > 0 {
		cfg.WSHost = wsListenAddr
	}

	if ctx.GlobalIsSet(utils.WSPortFlag.Name) {
		cfg.WSPort = ctx.GlobalInt(utils.WSPortFlag.Name)
	}

	//Producer Config
	if coinBase := ctx.GlobalString(utils.CoinBaseFlag.Name); len(coinBase) > 0 {
		cfg.CoinBase = coinBase
	}

	if ctx.GlobalIsSet(utils.MinerFlag.Name) {
		cfg.MinerEnabled = ctx.GlobalBool(utils.MinerFlag.Name)
	}

	if ctx.GlobalIsSet(utils.MinerIntervalFlag.Name) {
		cfg.MinerInterval = ctx.GlobalInt(utils.MinerIntervalFlag.Name)
	}

	//Log Level Config
	if logLevel := ctx.GlobalString(utils.LogLvlFlag.Name); len(logLevel) > 0 {
		cfg.LogLevel = logLevel
	}
}

func overrideNodeConfigs(ctx *cli.Context, cfg *node.Config) {

	if len(cfg.DataDir) == 0 || cfg.DataDir == "" {
		cfg.DataDir = common.DefaultDataDir()
	}

	if len(cfg.KeyStoreDir) == 0 || cfg.KeyStoreDir == "" {
		cfg.KeyStoreDir = cfg.DataDir
	}

	if ctx.GlobalBool(utils.DevNetFlag.Name) || cfg.NetID > 2 {
		cfg.NetSelect = "dev"
		//network override
		if cfg.NetID < 3 {
			cfg.NetID = 3
		}
		//dataDir override
		cfg.DataDir = filepath.Join(cfg.DataDir, "devdata")
		cfg.KeyStoreDir = filepath.Join(cfg.KeyStoreDir, "devdata", "wallet")
		//abs dataDir
		cfg.DataDirPathAbs()
		return
	}

	if ctx.GlobalBool(utils.TestNetFlag.Name) || cfg.NetID == 2 {
		cfg.NetSelect = "test"
		if cfg.NetID != 2 {
			cfg.NetID = 2
		}
		cfg.DataDir = filepath.Join(cfg.DataDir, "testdata")
		cfg.KeyStoreDir = filepath.Join(cfg.KeyStoreDir, "testdata", "wallet")
		cfg.DataDirPathAbs()
		return
	}

	if ctx.GlobalBool(utils.MainNetFlag.Name) || cfg.NetID == 1 {
		cfg.NetSelect = "main"
		if cfg.NetID != 1 {
			cfg.NetID = 1
		}
		cfg.DataDir = filepath.Join(cfg.DataDir, "maindata")
		cfg.KeyStoreDir = filepath.Join(cfg.KeyStoreDir, "maindata", "wallet")
		cfg.DataDirPathAbs()
		return
	}

	if len(cfg.LogLevel) == 0 {
		cfg.LogLevel = "info"
	}

}

func loadNodeConfigFromFile(ctx *cli.Context, cfg *node.Config) {

	// first read use settings
	if file := ctx.GlobalString(utils.ConfigFileFlag.Name); file != "" {

		if jsonConf, err := ioutil.ReadFile(file); err == nil {
			err = json.Unmarshal(jsonConf, &cfg)
			if err == nil {
				return
			}
			log.Warn("Cannot unmarshal the config file content", "error", err)
		}
	}

	// second read default settings
	log.Info(fmt.Sprintf("Will use the default config %v", defaultNodeConfigFileName))

	if jsonConf, err := ioutil.ReadFile(defaultNodeConfigFileName); err == nil {
		err = json.Unmarshal(jsonConf, &cfg)
		if err == nil {
			return
		}
		log.Warn("Cannot unmarshal the default config file content", "error", err)
	}
	log.Warn("Read the default config file content error, The program will skip here and continue processing")
}

func makeRunLogFile(cfg *node.Config) {
	if fileName, e := cfg.RunLogFile(); e == nil {

		logLevel, err := log15.LvlFromString(cfg.LogLevel)
		if err != nil {
			logLevel = log15.LvlInfo
		}
		log15.Root().SetHandler(
			log15.LvlFilterHandler(logLevel, log15.Must.FileHandler(fileName, log15.TerminalFormat())),
		)
	}
}
