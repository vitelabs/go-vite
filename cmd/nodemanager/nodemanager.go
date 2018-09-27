package nodemanager

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var (
	log = log15.New("module", "gvite/nodemanager")
)

type NodeManager struct {
	ctx    *cli.Context
	node   *node.Node
	logger log15.Logger
}

func New(ctx *cli.Context) NodeManager {
	return NodeManager{
		ctx:    ctx,
		node:   MakeFullNode(ctx),
		logger: log,
	}
}

func (nodeManager *NodeManager) Start() error {

	// Start up the node
	StartNode(nodeManager.node)

	return nil
}

func MakeFullNode(ctx *cli.Context) *node.Node {

	nodeConfig := MakeNodeConfig(ctx)

	node, err := node.New(nodeConfig)

	if err != nil {
		log.Error("Failed to create the node: %v", err)
	}
	return node
}

func MakeNodeConfig(ctx *cli.Context) *node.Config {

	cfg := node.DefaultNodeConfig

	// 1: Load config file.
	if file := ctx.GlobalString(utils.ConfigFileFlag.Name); file != "" {

		if jsonConf, err := ioutil.ReadFile(file); err == nil {
			err = json.Unmarshal(jsonConf, &cfg)
			if err != nil {
				log.Info("cannot unmarshal the config file content, will use the default config", "error", err)
			}
		} else {
			log.Info("cannot read the config file, will use the default config", "error", err)
		}
	}

	// 2: Apply flags, Overwrite the configuration file configuration
	mappingNodeConfig(ctx, &cfg)

	//3: Config log to file
	if fileName, e := cfg.NewRunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(fileName, log15.TerminalFormat())),
		)
	}

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
		cfg.Name = identity
		cfg.P2P.Name = identity
	}

	if ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		cfg.P2P.NetID = ctx.GlobalUint(utils.NetworkIdFlag.Name)
	}

	if ctx.GlobalIsSet(utils.MaxPeersFlag.Name) {
		cfg.P2P.MaxPeers = ctx.GlobalUint(utils.MaxPeersFlag.Name)
	}

	if ctx.GlobalIsSet(utils.MaxPendingPeersFlag.Name) {
		cfg.P2P.MaxPendingPeers = ctx.GlobalUint(utils.MaxPendingPeersFlag.Name)
	}

	if ctx.GlobalIsSet(utils.ListenPortFlag.Name) {
		cfg.P2P.Port = ctx.GlobalUint(utils.ListenPortFlag.Name)
	}

	if ctx.GlobalIsSet(utils.NodeKeyHexFlag.Name) {
		cfg.P2P.PrivateKey = ctx.GlobalString(utils.NodeKeyHexFlag.Name)
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

func StartNode(node *node.Node) {

	// Start the node
	if err := node.Start(); err != nil {
		log.Error("Error staring protocol node: %v", err)
	}

	// Listening event closes the node
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)
		<-c
		log.Info("Got interrupt, shutting down...")
		go node.Stop()
		for i := 10; i > 0; i-- {
			<-c
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}

	}()

	//TODO:why
	node.Wait()
}
