package utils

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"gopkg.in/urfave/cli.v1"
)

var (
	// Config settings
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "Json configuration file",
	}

	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "use for store all files",
		Value: DirectoryString{common.DefaultDataDir()}, // TODO Distinguish different environmental addresses
	}

	KeyStoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}

	// Network Settings
	TestNetFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Ropsten network: pre-configured proof-of-work test network",
	}

	DevNetFlag = cli.BoolFlag{
		Name:  "devnet",
		Usage: "Rinkeby network: pre-configured proof-of-authority dev network",
	}

	MainNetFlag = cli.BoolFlag{
		Name:  "mainnet",
		Usage: "Rinkeby network: pre-configured proof-of-authority prod network",
	}

	IdentityFlag = cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}
	NetworkIdFlag = cli.UintFlag{
		Name: "networkid", //mapping:p2p.NetID
		Usage: "Network identifier (integer," +
			" 1=MainNet," +
			" 2=TestNet," +
			" 3~12=DevNet,)",
		Value: config.GlobalConfig.NetID,
	}
	MaxPeersFlag = cli.UintFlag{
		Name:  "maxpeers", //mapping:p2p.MaxPeers
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: config.GlobalConfig.MaxPeers,
	}
	MaxPendingPeersFlag = cli.UintFlag{
		Name:  "maxpendpeers", //mapping:p2p.MaxPendingPeers
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: config.GlobalConfig.MaxPeers,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port", //mapping:p2p.Addr
		Usage: "Network listening port",
		Value: common.DefaultP2PPort,
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex", //mapping:p2p.PrivateKey
		Usage: "P2P node key as hex",
	}

	//IPC Settings
	IPCEnabledFlag = cli.BoolFlag{
		Name:  "ipc",
		Usage: "Enable the IPC-RPC server",
	}
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}

	//HTTP RPC Settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: common.DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: common.DefaultHTTPPort,
	}

	//WS Settings
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: common.DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
		Value: common.DefaultWSPort,
	}

	//Console Settings
	JSPathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement",
	}
	PreloadJSFlag = cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}
)

// This allows the use of the existing configuration functionality.
// When all flags are migrated this function can be removed and the existing
// configuration functionality must be changed that is uses local flags
func MigrateFlags(action func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, name := range ctx.FlagNames() {
			if ctx.IsSet(name) {
				ctx.GlobalSet(name, ctx.String(name))
			}
		}
		return action(ctx)
	}
}

// merge flags
func MergeFlags(flagsSet ...[]cli.Flag) []cli.Flag {

	mergeFlags := []cli.Flag{}

	for _, flags := range flagsSet {

		mergeFlags = append(mergeFlags, flags...)
	}
	return mergeFlags
}
