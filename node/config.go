package node

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/wallet"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// template：["broker1,broker2,...|topic",""]
	KafkaProducers []string `json:"KafkaProducers"`

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

	//producer
	CoinBase      string `json:"CoinBase"`
	MinerEnabled  bool   `json:"Miner"`
	MinerInterval int    `json:"MinerInterval"`

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

	HTTPCors      []string `json:"HTTPCors"`
	WSOrigins     []string `json:"WSOrigins"`
	PublicModules []string `json:"PublicModules"`

	//Log level
	LogLevel string `json:"LogLevel"`

	//VM
	VMTestEnabled bool `json:"VMTestEnabled"`

	//Net TODO: cmd after ？
	Single                 bool     `json:"Single"`
	FilePort               int      `json:"FilePort"`
	Topology               []string `json:"Topology"`
	TopologyTopic          string   `json:"TopologyTopic"`
	TopologyReportInterval int      `json:"TopologyReportInterval"`
}

func (c *Config) makeWalletConfig() *wallet.Config {
	return &wallet.Config{DataDir: c.KeyStoreDir}
}

func (c *Config) makeViteConfig() *config.Config {
	return &config.Config{
		Chain:    c.makeChainConfig(),
		P2P:      c.makeConfigP2P(),
		Producer: c.makeMinerConfig(),
		DataDir:  c.DataDir,
		Net:      c.makeNetConfig(),
		Vm:       c.makeVmConfig(),
		LogLevel: c.LogLevel,
	}
}

func (c *Config) makeNetConfig() *config.Net {
	return &config.Net{
		Single:   c.Single,
		FilePort: uint16(c.FilePort),
		Topology: c.Topology,
		Topic:    c.TopologyTopic,
		Interval: int64(c.TopologyReportInterval),
	}
}

func (c *Config) makeVmConfig() *config.Vm {
	return &config.Vm{
		IsVmTest: c.VMTestEnabled,
	}
}

func (c *Config) makeMinerConfig() *config.Producer {
	return &config.Producer{
		Producer: c.MinerEnabled,
		Coinbase: c.CoinBase,
	}
}

func (c *Config) makeConfigP2P() *config.P2P {
	return &config.P2P{
		Name:                 c.Identity,
		NetID:                c.NetID,
		MaxPeers:             c.MaxPeers,
		MaxPendingPeers:      c.MaxPendingPeers,
		MaxPassivePeersRatio: c.MaxPassivePeersRatio,
		Port:                 c.Port,
		Datadir:              filepath.Join(c.DataDir, p2p.Dirname),
		PrivateKey:           c.PrivateKey,
		BootNodes:            c.BootNodes,
	}
}

func (c *Config) makeP2PConfig() *p2p.Config {
	return &p2p.Config{
		Name:            c.Identity,
		NetID:           p2p.NetworkID(c.NetID),
		MaxPeers:        c.MaxPeers,
		MaxPendingPeers: c.MaxPendingPeers,
		MaxInboundRatio: c.MaxPassivePeersRatio,
		Port:            c.Port,
		DataDir:         filepath.Join(c.DataDir, p2p.Dirname),
		PrivateKey:      c.GetPrivateKey(),
		//Protocols:nil,
		BootNodes:   c.BootNodes,
		StaticNodes: c.StaticNodes,
		//KafKa:nil,
	}
}

func (c *Config) makeChainConfig() *config.Chain {

	if len(c.KafkaProducers) == 0 {
		return &config.Chain{
			KafkaProducers: nil,
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
	return filepath.Join(c.DataDir, "runlog")
}

func (c *Config) RunLogFile() (string, error) {
	filename := time.Now().Format("2006-01-02") + ".log"
	if err := os.MkdirAll(c.RunLogDir(), 0777); err != nil {
		return "", err
	}
	return filepath.Join(c.RunLogDir(), filename), nil

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
