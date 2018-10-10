package nodemanager

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/utils"
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
	log.Info(fmt.Sprintf("nodeConfig info: %v", nodeConfig))
	// 2: New Node
	node, err := node.New(nodeConfig)

	if err != nil {
		log.Error("Failed to create the node: %v", err)
	}
	return node
}

func (maker FullNodeMaker) MakeNodeConfig(ctx *cli.Context) *node.Config {

	cfg := node.DefaultNodeConfig

	// 1: Load config file.
	loadNodeConfigFromFile(ctx, &cfg)

	// 2: Apply flags, Overwrite the configuration file configuration
	mappingNodeConfig(ctx, &cfg)

	// 3: Override any default configs for hard coded networks.
	overrideDefaultConfigs(ctx, &cfg)

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
	if ctx.GlobalIsSet(utils.KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.GlobalString(utils.KeyStoreDirFlag.Name)
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

	if ctx.GlobalIsSet(utils.NodeKeyHexFlag.Name) {
		cfg.SetPrivateKey(ctx.GlobalString(utils.NodeKeyHexFlag.Name))
	}

	//Ipc Config
	cfg.IPCEnabled = ctx.GlobalBool(utils.IPCEnabledFlag.Name)

	if ctx.GlobalIsSet(utils.IPCPathFlag.Name) {
		cfg.IPCPath = ctx.GlobalString(utils.IPCPathFlag.Name)
	}

	//Http Config
	cfg.RPCEnabled = ctx.GlobalBool(utils.RPCEnabledFlag.Name)

	if ctx.GlobalIsSet(utils.RPCListenAddrFlag.Name) {
		cfg.HttpHost = ctx.GlobalString(utils.RPCListenAddrFlag.Name)
	}

	if ctx.GlobalIsSet(utils.RPCPortFlag.Name) {
		cfg.HttpPort = ctx.GlobalInt(utils.RPCPortFlag.Name)
	}

	//WS Config
	cfg.WSEnabled = ctx.GlobalBool(utils.WSEnabledFlag.Name)

	if ctx.GlobalIsSet(utils.WSListenAddrFlag.Name) {
		cfg.WSHost = ctx.GlobalString(utils.WSListenAddrFlag.Name)
	}

	if ctx.GlobalIsSet(utils.WSPortFlag.Name) {
		cfg.WSPort = ctx.GlobalInt(utils.WSPortFlag.Name)
	}

}

func overrideDefaultConfigs(ctx *cli.Context, cfg *node.Config) {

	if ctx.GlobalBool(utils.DevNetFlag.Name) {
		cfg.NetSelect = "dev"
		//network override
		if cfg.NetID < 3 {
			cfg.NetID = 3
		}
		//dataDir override
		cfg.DataDir = filepath.Join(cfg.DataDir, "devdata")
		//abs dataDir
		cfg.DataDirPathAbs()
		return
	}

	if ctx.GlobalBool(utils.TestNetFlag.Name) {
		cfg.NetSelect = "test"
		if cfg.NetID != 2 {
			cfg.NetID = 2
		}
		cfg.DataDir = filepath.Join(cfg.DataDir, "testdata")
		cfg.DataDirPathAbs()
		return
	}

	if ctx.GlobalBool(utils.MainNetFlag.Name) {
		cfg.NetSelect = "main"
		if cfg.NetID != 1 {
			cfg.NetID = 1
		}
		cfg.DataDir = filepath.Join(cfg.DataDir, "viteisbest")
		cfg.DataDirPathAbs()
		return
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
			log.Warn("cannot unmarshal the config file content", "error", err)
		}
	}

	// second read default settings
	log.Info(fmt.Sprintf("will use the default config %v", defaultNodeConfigFileName))

	if jsonConf, err := ioutil.ReadFile(defaultNodeConfigFileName); err == nil {
		err = json.Unmarshal(jsonConf, &cfg)
		if err == nil {
			return
		}
		log.Warn("cannot unmarshal the default config file content", "error", err)
	}
	log.Warn("read the default config file content error, The program will skip here and continue processing")
}

func makeRunLogFile(cfg *node.Config) {
	if fileName, e := cfg.RunLogFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(fileName, log15.TerminalFormat())),
		)
	}
}
