package nodeconfig

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/v2/common"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
)

type Config struct {
	NetSelect string

	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// chain
	LedgerGcRetain uint64          `json:"LedgerGcRetain"`
	LedgerGc       *bool           `json:"LedgerGc"`
	OpenPlugins    *bool           `json:"OpenPlugins"`
	VmLogWhiteList []types.Address `json:"vmLogWhiteList"` // contract address white list which save VM logs
	VmLogAll       *bool           `json:"vmLogAll"`       // save all VM logs, it will cost more disk space

	// genesis
	GenesisFile string `json:"GenesisFile"`

	// net
	Single             bool
	ListenInterface    string
	Port               int
	FilePort           int
	PublicAddress      string
	FilePublicAddress  string
	Identity           string
	NetID              int
	PeerKey            string `json:"PrivateKey"`
	Discover           bool
	MaxPeers           int
	MinPeers           int
	MaxInboundRatio    int
	MaxPendingPeers    int
	BootNodes          []string
	BootSeeds          []string
	StaticNodes        []string
	AccessControl      string
	AccessAllowKeys    []string
	AccessDenyKeys     []string
	BlackBlockHashList []string // from high to low, like: "xxxxxx-11111"
	WhiteBlockList     []string // from high to low, like: "xxxxxx-10001"
	ForwardStrategy    string

	//producer
	EntropyStorePath     string `json:"EntropyStorePath"`
	EntropyStorePassword string `json:"EntropyStorePassword"`
	CoinBase             string `json:"CoinBase"`
	MinerEnabled         bool   `json:"Miner"`

	//rpc
	RPCEnabled  bool  `json:"RPCEnabled"`
	IPCEnabled  bool  `json:"IPCEnabled"`
	WSEnabled   bool  `json:"WSEnabled"`
	TxDexEnable *bool `json:"TxDexEnable"`

	IPCPath          string   `json:"IPCPath"`
	HttpHost         string   `json:"HttpHost"`
	HttpPort         int      `json:"HttpPort"`
	HttpVirtualHosts []string `json:"HttpVirtualHosts"`
	WSHost           string   `json:"WSHost"`
	WSPort           int      `json:"WSPort"`
	PrivateHttpPort  int      `json:"PrivateHttpPort"`

	HTTPCors            []string `json:"HTTPCors"`
	WSOrigins           []string `json:"WSOrigins"`
	PublicModules       []string `json:"PublicModules"`
	WSExposeAll         bool     `json:"WSExposeAll"`
	HttpExposeAll       bool     `json:"HttpExposeAll"`
	TestTokenHexPrivKey string   `json:"TestTokenHexPrivKey"`
	TestTokenTti        string   `json:"TestTokenTti"`

	PowServerUrl string `json:"PowServerUrl"`

	//Log level
	LogLevel    string `json:"LogLevel"`
	ErrorLogDir string `json:"ErrorLogDir"`

	//VM
	VMTestEnabled         bool `json:"VMTestEnabled"`
	VMTestParamEnabled    bool `json:"VMTestParamEnabled"`
	QuotaTestParamEnabled bool `json:"QuotaTestParamEnabled"`
	VMDebug               bool `json:"VMDebug"`

	// subscribe
	SubscribeEnabled bool `json:"SubscribeEnabled"`

	// dashboard
	DashboardTargetURL string

	// reward
	RewardAddr string `json:"RewardAddr"`

	//metrics
	MetricsEnable    *bool   `json:"MetricsEnable"`
	InfluxDBEnable   *bool   `json:"InfluxDBEnable"`
	InfluxDBEndpoint *string `json:"InfluxDBEndpoint"`
	InfluxDBDatabase *string `json:"InfluxDBDatabase"`
	InfluxDBUsername *string `json:"InfluxDBUsername"`
	InfluxDBPassword *string `json:"InfluxDBPassword"`
	InfluxDBHostTag  *string `json:"InfluxDBHostTag"`
}

func (c *Config) MakeWalletConfig() *config.Wallet {
	return &config.Wallet{DataDir: c.KeyStoreDir}
}

func (c *Config) MakeViteConfig() *config.Config {
	return &config.Config{
		Chain:      c.MakeChainConfig(),
		Producer:   c.MakeMinerConfig(),
		DataDir:    c.DataDir,
		Net:        c.MakeNetConfig(),
		Vm:         c.MakeVmConfig(),
		Subscribe:  c.MakeSubscribeConfig(),
		NodeReward: c.MakeRewardConfig(),
		Genesis:    config.MakeGenesisConfig(c.GenesisFile),
		LogLevel:   c.LogLevel,
	}
}

func (c *Config) MakeNetConfig() *config.Net {
	datadir := filepath.Join(c.DataDir, config.DefaultNetDirName)

	return &config.Net{
		Single:             c.Single,
		Name:               c.Identity,
		NetID:              c.NetID,
		ListenInterface:    c.ListenInterface,
		Port:               c.Port,
		FilePort:           c.FilePort,
		PublicAddress:      c.PublicAddress,
		FilePublicAddress:  c.FilePublicAddress,
		DataDir:            datadir,
		PeerKey:            c.PeerKey,
		Discover:           c.Discover,
		BootNodes:          c.BootNodes,
		BootSeeds:          c.BootSeeds,
		StaticNodes:        c.StaticNodes,
		MaxPeers:           c.MaxPeers,
		MaxInboundRatio:    c.MaxInboundRatio,
		MinPeers:           c.MinPeers,
		MaxPendingPeers:    c.MaxPendingPeers,
		ForwardStrategy:    c.ForwardStrategy,
		AccessControl:      c.AccessControl,
		AccessAllowKeys:    c.AccessAllowKeys,
		AccessDenyKeys:     c.AccessDenyKeys,
		BlackBlockHashList: c.BlackBlockHashList,
		WhiteBlockList:     c.WhiteBlockList,
		MineKey:            nil,
	}
}

func (c *Config) MakeRewardConfig() *config.NodeReward {
	return &config.NodeReward{
		RewardAddr: c.RewardAddr,
		Name:       c.Identity,
	}
}

func (c *Config) MakeVmConfig() *config.Vm {
	return &config.Vm{
		IsVmTest:            c.VMTestEnabled,
		IsUseVmTestParam:    c.VMTestParamEnabled,
		IsUseQuotaTestParam: c.QuotaTestParamEnabled,
		IsVmDebug:           c.VMDebug,
	}
}

func (c *Config) MakeSubscribeConfig() *config.Subscribe {
	return &config.Subscribe{
		IsSubscribe: c.SubscribeEnabled,
	}
}
func (c *Config) MakeMinerConfig() *config.Producer {
	cfg := &config.Producer{
		Producer:                c.MinerEnabled,
		Coinbase:                c.CoinBase,
		EntropyStorePath:        c.EntropyStorePath,
		VirtualSnapshotVerifier: false,
	}
	err := cfg.Parse()
	if err != nil {
		panic(err)
	}
	return cfg
}

func (c *Config) MakeChainConfig() *config.Chain {

	// is open ledger gc
	ledgerGc := true
	if c.LedgerGc != nil {
		ledgerGc = *c.LedgerGc
	}
	// is open plugins
	openPlugins := false
	if c.OpenPlugins != nil {
		openPlugins = *c.OpenPlugins
	}

	// save all VM logs, it will cost more disk space
	vmLogAll := false
	if c.VmLogAll != nil {
		vmLogAll = *c.VmLogAll
	}
	return &config.Chain{
		LedgerGcRetain: c.LedgerGcRetain,
		LedgerGc:       ledgerGc,
		OpenPlugins:    openPlugins,
		VmLogWhiteList: c.VmLogWhiteList,
		VmLogAll:       vmLogAll,
	}
}

func (c *Config) HTTPEndpoint() string {
	if c.HttpHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HttpHost, c.HttpPort)
}

func (c *Config) WSEndpoint() string {
	if c.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)
}

func (c *Config) PrivateHTTPEndpoint() string {
	if c.PrivateHttpPort == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", common.DefaultHTTPHost, c.PrivateHttpPort)
}

func (c *Config) SetPrivateKey(privateKey string) {
	c.PeerKey = privateKey
}

func (c *Config) GetPrivateKey() ed25519.PrivateKey {
	if c.PeerKey != "" {
		privateKey, err := hex.DecodeString(c.PeerKey)
		if err == nil {
			return ed25519.PrivateKey(privateKey)
		}
	}

	return nil
}

func (c *Config) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IPCPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IPCPath, `\\.\pipe\`) {
			return c.IPCPath
		}
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IPCPath) == c.IPCPath {
		if c.DataDir == "" {
			return filepath.Join(os.TempDir(), c.IPCPath)
		}
		return filepath.Join(c.DataDir, c.IPCPath)
	}
	return c.IPCPath
}

func (c *Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog", time.Now().Format("2006-01-02T15-04"))
}

// resolve the dataDir so future changes to the current working directory don't affect the node
func (c *Config) DataDirPathAbs() error {
	if c.DataDir != "" {
		absDataDir, err := filepath.Abs(c.DataDir)
		if err != nil {
			return err
		}
		c.DataDir = absDataDir
	}

	if c.KeyStoreDir != "" {
		absKeyStoreDir, err := filepath.Abs(c.KeyStoreDir)
		if err != nil {
			return err
		}
		c.KeyStoreDir = absKeyStoreDir
	}
	return nil
}

func (c *Config) ParseFromFile(filename string) error {
	if jsonConf, err := ioutil.ReadFile(filename); err == nil {
		err = json.Unmarshal(jsonConf, &c)
		if err == nil {
			return nil
		}
		return err
	}
	return fmt.Errorf("read config file %s error", filename)
}
