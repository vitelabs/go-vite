package node

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vitelabs/go-vite/config/biz"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/network"
	"github.com/vitelabs/go-vite/wallet"
)

type Config struct {
	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// template：["broker1,broker2,...|topic",""]
	KafkaProducers []string `json:"KafkaProducers"`

	// chain
	OpenBlackBlock bool   `json:"OpenBlackBlock"`
	LedgerGcRetain uint64 `json:"LedgerGcRetain"`
	GenesisFile    string `json:"GenesisFile"`
	LedgerGc       bool   `json:"LedgerGc"`

	// p2p
	NetSelect            string
	Identity             string   `json:"Identity"`
	PrivateKey           string   `json:"PrivateKey"`
	MaxPeers             uint     `json:"MaxPeers"`
	MaxPassivePeersRatio uint     `json:"MaxPassivePeersRatio"`
	MaxPendingPeers      uint     `json:"MaxPendingPeers"`
	BootNodes            []string `json:"BootNodes"`
	StaticNodes          []string `json:"StaticNodes"`
	Port                 uint     `json:"Port"`
	NetID                uint     `json:"NetID"`
	Discovery            bool     `json:"Discovery"`

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

	//Net TODO: cmd after ？
	Single                 bool     `json:"Single"`
	FilePort               int      `json:"FilePort"`
	Topology               []string `json:"Topology"`
	TopologyTopic          string   `json:"TopologyTopic"`
	TopologyReportInterval int      `json:"TopologyReportInterval"`
	TopoDisabled           bool     `json:"TopoDisabled"`
	DashboardTargetURL     string

	// reward
	RewardAddr string `json:"RewardAddr"`

	//metrics
	MetricsEnable         bool `json:"MetricsEnable"`
	MetricsInfluxDBEnable bool `json:"MetricsInfluxDBEnable"`
}

func (c *Config) makeWalletConfig() *wallet.Config {
	return &wallet.Config{DataDir: c.KeyStoreDir}
}

func (c *Config) makeViteConfig() *config.Config {
	return &config.Config{
		Chain:    c.makeChainConfig(),
		Producer: c.makeMinerConfig(),
		DataDir:  c.DataDir,
		Net:      c.makeNetConfig(),
		Vm:       c.makeVmConfig(),
		Reward:   c.makeRewardConfig(),
		LogLevel: c.LogLevel,
	}
}

func (c *Config) makeNetConfig() *config.Net {
	return &config.Net{
		Single:       c.Single,
		FilePort:     uint16(c.FilePort),
		Topology:     c.Topology,
		Topic:        c.TopologyTopic,
		Interval:     int64(c.TopologyReportInterval),
		TopoDisabled: c.TopoDisabled,
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
	}
}

func (c *Config) makeMinerConfig() *config.Producer {
	return &config.Producer{
		Producer:         c.MinerEnabled,
		Coinbase:         c.CoinBase,
		EntropyStorePath: c.EntropyStorePath,
	}
}

func (c *Config) makeP2PConfig() *p2p.Config {
	return &p2p.Config{
		Name:            c.Identity,
		NetID:           network.ID(c.NetID),
		MaxPeers:        c.MaxPeers,
		MaxPendingPeers: c.MaxPendingPeers,
		MaxInboundRatio: c.MaxPassivePeersRatio,
		Port:            c.Port,
		DataDir:         filepath.Join(c.DataDir, p2p.Dirname),
		PrivateKey:      c.GetPrivateKey(),
		BootNodes:       c.BootNodes,
		StaticNodes:     c.StaticNodes,
		Discovery:       c.Discovery,
	}
}

func (c *Config) makeChainConfig() *config.Chain {

	if len(c.KafkaProducers) == 0 {
		return &config.Chain{
			KafkaProducers: nil,
			OpenBlackBlock: c.OpenBlackBlock,
			LedgerGcRetain: c.LedgerGcRetain,
			GenesisFile:    c.GenesisFile,
			LedgerGc:       true,
		}
	}

	// init kafkaProducers
	kafkaProducers := make([]*config.KafkaProducer, len(c.KafkaProducers))

	for i, kafkaProducer := range c.KafkaProducers {
		splitKafkaProducer := strings.Split(kafkaProducer, "|")
		if len(splitKafkaProducer) != 2 {
			log.Warn(fmt.Sprintf("KafkaProducers is setting error，The program will skip here and continue processing"))
			goto END
		}

		splitKafkaBroker := strings.Split(splitKafkaProducer[0], ",")
		if len(splitKafkaBroker) == 0 {
			log.Warn(fmt.Sprintf("KafkaProducers is setting error，The program will skip here and continue processing"))
			goto END
		}

		kafkaProducers[i] = &config.KafkaProducer{
			BrokerList: splitKafkaBroker,
			Topic:      splitKafkaProducer[1],
		}
	}
END:
	return &config.Chain{
		KafkaProducers: kafkaProducers,
		OpenBlackBlock: c.OpenBlackBlock,
		LedgerGcRetain: c.LedgerGcRetain,
		GenesisFile:    c.GenesisFile,
		LedgerGc:       c.LedgerGc,
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
	c.PrivateKey = privateKey
}

func (c *Config) GetPrivateKey() ed25519.PrivateKey {

	if c.PrivateKey != "" {
		privateKey, err := hex.DecodeString(c.PrivateKey)
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

//resolve the dataDir so future changes to the current working directory don't affect the node
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
