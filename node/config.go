package node

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vitelabs/go-vite/p2p/discovery"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/config/biz"
	"github.com/vitelabs/go-vite/config/gen"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/metrics"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/wallet"
)

type Config struct {
	NetSelect string

	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// template：["broker1,broker2,...|topic",""]
	KafkaProducers []string `json:"KafkaProducers"`

	// chain
	LedgerGcRetain uint64 `json:"LedgerGcRetain"`
	LedgerGc       *bool  `json:"LedgerGc"`
	OpenPlugins    *bool  `json:"OpenPlugins"`

	// genesis
	GenesisFile string `json:"GenesisFile"`

	// p2p
	Identity        string   `json:"Identity"`
	PeerKey         string   `json:"PeerKey"`
	PrivateKey      string   `json:"PrivateKey"`
	MaxPeers        int      `json:"MaxPeers"`
	MinPeers        int      `json:"MinPeers"`
	MaxInboundRatio int      `json:"MaxInboundRatio"`
	MaxPendingPeers int      `json:"MaxPendingPeers"`
	BootNodes       []string `json:"BootNodes"`
	BootSeeds       []string `json:"BootSeeds"`
	StaticNodes     []string `json:"StaticNodes"`
	ListenInterface string   `json:"ListenInterface"`
	Port            int      `json:"Port"`
	ListenAddress   string   `json:"ListenAddress"`
	PublicAddress   string   `json:"PublicAddress"`
	NetID           int      `json:"NetID"`
	Discover        bool     `json:"Discover"`

	//producer
	EntropyStorePath     string `json:"EntropyStorePath"`
	EntropyStorePassword string `json:"EntropyStorePassword"`
	CoinBase             string `json:"CoinBase"`
	MinerEnabled         bool   `json:"Miner"`
	MinerInterval        int    `json:"MinerInterval"`

	//rpc
	RPCEnabled bool `json:"RPCEnabled"`
	IPCEnabled bool `json:"IPCEnabled"`
	WSEnabled  bool `json:"WSEnabled"`

	IPCPath          string   `json:"IPCPath"`
	HttpHost         string   `json:"HttpHost"`
	HttpPort         int      `json:"HttpPort"`
	HttpVirtualHosts []string `json:"HttpVirtualHosts"`
	WSHost           string   `json:"WSHost"`
	WSPort           int      `json:"WSPort"`

	HTTPCors            []string `json:"HTTPCors"`
	WSOrigins           []string `json:"WSOrigins"`
	PublicModules       []string `json:"PublicModules"`
	WSExposeAll         bool     `json:"WSExposeAll"`
	HttpExposeAll       bool     `json:"HttpExposeAll"`
	TestTokenHexPrivKey string   `json:"TestTokenHexPrivKey"`
	TestTokenTti        string   `json:"TestTokenTti"`

	PowServerUrl string `json:"PowServerUrl”`

	//Log level
	LogLevel    string `json:"LogLevel"`
	ErrorLogDir string `json:"ErrorLogDir"`

	//VM
	VMTestEnabled      bool `json:"VMTestEnabled"`
	VMTestParamEnabled bool `json:"VMTestParamEnabled"`
	VMDebug            bool `json:"VMDebug"`

	// subscribe
	SubscribeEnabled bool `json:"SubscribeEnabled"`

	// net
	Single            bool   `json:"Single"`
	FilePort          int    `json:"FilePort"`
	FileListenAddress string `json:"FileListenAddress"`
	FilePublicAddress string `json:"FileAddress"`
	ForwardStrategy   string `json:"ForwardStrategy"`

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

func (c *Config) makeWalletConfig() *wallet.Config {
	return &wallet.Config{DataDir: c.KeyStoreDir}
}

func (c *Config) makeViteConfig() *config.Config {
	return &config.Config{
		Chain:     c.makeChainConfig(),
		Producer:  c.makeMinerConfig(),
		DataDir:   c.DataDir,
		Net:       c.makeNetConfig(),
		Vm:        c.makeVmConfig(),
		Subscribe: c.makeSubscribeConfig(),
		Reward:    c.makeRewardConfig(),
		Genesis:   config_gen.MakeGenesisConfig(c.GenesisFile),
		LogLevel:  c.LogLevel,
	}
}

func (c *Config) makeNetConfig() *config.Net {
	var fileListenAddress = c.FileListenAddress
	if fileListenAddress == "" {
		fileListenAddress = c.ListenInterface + ":" + strconv.Itoa(c.FilePort)
	}

	return &config.Net{
		Single:            c.Single,
		FileListenAddress: fileListenAddress,
		FilePublicAddress: c.FilePublicAddress,
		FilePort:          c.FilePort,
	}
}

func (c *Config) makeRewardConfig() *biz.Reward {
	return &biz.Reward{
		RewardAddr: c.RewardAddr,
		Name:       c.Identity,
	}
}

func (c *Config) makeVmConfig() *config.Vm {
	return &config.Vm{
		IsVmTest:         c.VMTestEnabled,
		IsUseVmTestParam: c.VMTestParamEnabled,
		IsVmDebug:        c.VMDebug,
	}
}

func (c *Config) makeSubscribeConfig() *config.Subscribe {
	return &config.Subscribe{
		IsSubscribe: c.SubscribeEnabled,
	}
}

func (c *Config) makeMetricsConfig() *metrics.Config {
	mc := &metrics.Config{
		IsEnable:         false,
		IsInfluxDBEnable: false,
		InfluxDBInfo:     nil,
	}
	if c.MetricsEnable != nil && *c.MetricsEnable == true {
		mc.IsEnable = true
		if c.InfluxDBEnable != nil && *c.InfluxDBEnable == true &&
			c.InfluxDBEndpoint != nil && len(*c.InfluxDBEndpoint) > 0 &&
			(c.InfluxDBEndpoint != nil && c.InfluxDBDatabase != nil && c.InfluxDBPassword != nil && c.InfluxDBHostTag != nil) {
			mc.IsInfluxDBEnable = true
			mc.InfluxDBInfo = &metrics.InfluxDBConfig{
				Endpoint: *c.InfluxDBEndpoint,
				Database: *c.InfluxDBDatabase,
				Username: *c.InfluxDBUsername,
				Password: *c.InfluxDBPassword,
				HostTag:  *c.InfluxDBHostTag,
			}
		}
	}

	return mc
}

func (c *Config) makeMinerConfig() *config.Producer {
	return &config.Producer{
		Producer:         c.MinerEnabled,
		Coinbase:         c.CoinBase,
		EntropyStorePath: c.EntropyStorePath,
	}
}

func (c *Config) makeP2PConfig() (cfg *p2p.Config, err error) {
	var listenAddress = c.ListenAddress
	if listenAddress == "" {
		listenAddress = c.ListenInterface + ":" + strconv.Itoa(c.Port)
	}

	p2pDataDir := filepath.Join(c.DataDir, p2p.DirName)

	// create data dir
	if err = os.MkdirAll(p2pDataDir, 0700); err != nil {
		return nil, err
	}

	peerKey := c.PeerKey
	if peerKey == "" {
		peerKey = c.PrivateKey
	}

	cfg = &p2p.Config{
		Config: &discovery.Config{
			ListenAddress: listenAddress,
			PublicAddress: c.PublicAddress,
			DataDir:       p2pDataDir,
			PeerKey:       c.PeerKey,
			BootNodes:     c.BootNodes,
			BootSeeds:     c.BootSeeds,
			NetID:         c.NetID,
		},
		Discover:        c.Discover,
		Name:            c.Identity,
		MaxPeers:        c.MaxPeers,
		MaxInboundRatio: c.MaxInboundRatio,
		MinPeers:        c.MinPeers,
		MaxPendingPeers: c.MaxPendingPeers,
		StaticNodes:     c.StaticNodes,
	}

	err = cfg.Ensure()

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) makeChainConfig() *config.Chain {

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

	return &config.Chain{
		LedgerGcRetain: c.LedgerGcRetain,
		LedgerGc:       ledgerGc,
		OpenPlugins:    openPlugins,
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

func (c *Config) RunLogHandler() log15.Handler {
	filename := "vite.log"
	logger := common.MakeDefaultLogger(filepath.Join(c.RunLogDir(), filename))
	return log15.StreamHandler(logger, log15.LogfmtFormat())
}

func (c *Config) RunErrorLogHandler() log15.Handler {
	filename := "vite.error.log"
	logger := common.MakeDefaultLogger(filepath.Join(c.RunLogDir(), "error", filename))
	return log15.StreamHandler(logger, log15.LogfmtFormat())
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
