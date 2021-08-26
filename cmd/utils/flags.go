package utils

import (
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
	}

	KeyStoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}

	// Network Settings
	TestNetFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "test network, networkdid==2",
	}

	DevNetFlag = cli.BoolFlag{
		Name:  "devnet",
		Usage: "develop network, networkdid>=3",
	}

	MainNetFlag = cli.BoolFlag{
		Name:  "mainnet",
		Usage: "main network, networkdid==1",
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
	}
	MaxPeersFlag = cli.UintFlag{
		Name:  "maxpeers", //mapping:p2p.MaxPeers
		Usage: "Maximum number of network peers (network disabled if set to 0)",
	}
	MaxPendingPeersFlag = cli.UintFlag{
		Name:  "maxpendpeers", //mapping:p2p.MaxPendingPeers
		Usage: "Maximum number of db connection attempts (defaults used if set to 0)",
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port", //mapping:p2p.Addr
		Usage: "Network listening port",
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex", //mapping:p2p.PeerKey
		Usage: "P2P node key as hex",
	}
	DiscoveryFlag = cli.StringFlag{
		Name:  "discovery", //mapping:p2p.Discovery
		Usage: "enable p2p discovery or not",
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
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
	}

	//WS Settings
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
	}
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
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

	//Producer
	MinerFlag = cli.BoolFlag{
		Name:  "miner",
		Usage: "Enable the Miner",
	}

	CoinBaseFlag = cli.StringFlag{
		Name:  "coinbase",
		Usage: "Coinbase is an address into which the rewards for the SuperNode produce snapshot-block",
	}

	MinerIntervalFlag = cli.IntFlag{
		Name:  "minerinterval",
		Usage: "Miner Interval(unit: second)",
	}

	//Log Lvl
	LogLvlFlag = cli.StringFlag{
		Name:  "loglevel",
		Usage: "log level (info,eror,warn,dbug)",
	}

	//VM
	VMTestFlag = cli.BoolFlag{
		Name:  "vmtest",
		Usage: "Enable the VM test ",
	}
	VMTestParamFlag = cli.BoolFlag{
		Name:  "vmtestparam",
		Usage: "Enable the VM test params ",
	}
	QuotaTestParamFlag = cli.BoolFlag{
		Name:  "quotatestparam",
		Usage: "Enable the VM quota test params ",
	}
	VMDebugFlag = cli.BoolFlag{
		Name:  "vmdebug",
		Usage: "Enable VM debug",
	}

	// Subscribe
	SubscribeFlag = cli.BoolFlag{
		Name:  "subscribe",
		Usage: "Enable Subscribe",
	}

	// Ledger
	LedgerDeleteToHeight = cli.Uint64Flag{
		Name:  "del",
		Usage: "Delete to height",
	}

	// Trie
	RecoverTrieFlag = cli.BoolFlag{
		Name:  "trie",
		Usage: "Recover trie",
	}

	// Export sb height
	ExportSbHeightFlags = cli.Uint64Flag{
		Name:  "sbHeight",
		Usage: "The snapshot block height",
	}

	//Net
	SingleFlag = cli.BoolFlag{
		Name:  "single",
		Usage: "Enable the NodeServer single ",
	}

	FilePortFlag = cli.IntFlag{
		Name:  "fileport",
		Usage: "File transfer listening port",
	}

	//Stat
	PProfEnabledFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable chain performance analysis tool, you can visit the address[http://localhost:8080/debug/pprof]",
	}

	PProfPortFlag = cli.UintFlag{
		Name:  "pprofport",
		Usage: "pporof visit `port`, you can visit the address[http://localhost:`port`/debug/pprof]",
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
	flagMap := make(map[string]cli.Flag)
	for _, flags := range flagsSet {
		for _, flag := range flags {
			flagMap[flag.GetName()] = flag
		}
	}

	var flags []cli.Flag
	for _, flag := range flagMap {
		flags = append(flags, flag)
	}
	return flags
}

var (
	ConfigFlags = []cli.Flag{
		ConfigFileFlag,
	}
	//general
	GeneralFlags = []cli.Flag{
		DataDirFlag,
		KeyStoreDirFlag,
	}

	//p2p
	P2pFlags = []cli.Flag{
		DevNetFlag,
		TestNetFlag,
		MainNetFlag,
		IdentityFlag,
		NetworkIdFlag,
		MaxPeersFlag,
		MaxPendingPeersFlag,
		ListenPortFlag,
		NodeKeyHexFlag,
		DiscoveryFlag,
	}

	//IPC
	IpcFlags = []cli.Flag{
		IPCEnabledFlag,
		IPCPathFlag,
	}

	//HTTP RPC
	HttpFlags = []cli.Flag{
		RPCEnabledFlag,
		RPCListenAddrFlag,
		RPCPortFlag,
	}

	//WS
	WsFlags = []cli.Flag{
		WSEnabledFlag,
		WSListenAddrFlag,
		WSPortFlag,
	}

	//Console
	ConsoleFlags = []cli.Flag{
		JSPathFlag,
		ExecFlag,
		PreloadJSFlag,
	}

	//Producer
	ProducerFlags = []cli.Flag{
		MinerFlag,
		CoinBaseFlag,
		MinerIntervalFlag,
	}

	//Log
	LogFlags = []cli.Flag{
		LogLvlFlag,
	}

	//VM
	VmFlags = []cli.Flag{
		VMTestFlag,
		VMTestParamFlag,
	}

	//Net
	NetFlags = []cli.Flag{
		SingleFlag,
		FilePortFlag,
	}

	//Stat
	StatFlags = []cli.Flag{
		PProfEnabledFlag,
		PProfPortFlag,
	}

	// Ledger
	LedgerFlags = []cli.Flag{
		LedgerDeleteToHeight,
		RecoverTrieFlag,
	}

	// Export
	ExportFlags = []cli.Flag{
		ExportSbHeightFlags,
	}

	// Load
	LoadLedgerFlags = []cli.Flag{
		// Load From Directory
		cli.StringFlag{
			Name:  "fromDir",
			Usage: "from directory",
		},
	}
)
