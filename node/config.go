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

	"encoding/json"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/network"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
)

type Config struct {
	DataDir string `json:"DataDir"`

	KeyStoreDir string `json:"KeyStoreDir"`

	// template：["broker1,broker2,...|topic",""]
	KafkaProducers []string `json:"KafkaProducers"`

	// chain
	OpenBlackBlock bool   `json:"OpenBlackBlock"`
	LedgerGcRetain uint64 `json:"LedgerGcRetain"`
	LedgerGc       bool   `json:"LedgerGc"`

	// genesis
	GenesisFile string `json:"GenesisFile"`

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
	VMDebug            bool `json:"VMDebug"`

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
		Genesis:  c.makeGenesisConfig(),
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
		IsVmDebug:        c.VMDebug,
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
		PeerKey:         c.GetPrivateKey(),
		BootNodes:       c.BootNodes,
		StaticNodes:     c.StaticNodes,
		Discovery:       c.Discovery,
	}
}

func (c *Config) makeForkPointsConfig(genesisConfig *config.Genesis) *config.ForkPoints {
	forkPoints := &config.ForkPoints{}

	if genesisConfig != nil && genesisConfig.ForkPoints != nil {
		forkPoints = genesisConfig.ForkPoints
	}

	if forkPoints.Vite1 == nil {
		forkPoints.Vite1 = &config.ForkPoint{}
	}

	return forkPoints
}

func (c *Config) makeGenesisConfig() *config.Genesis {
	defaultGenesisAccountAddress, _ := types.HexToAddress("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")
	var defaultBlockProducers []types.Address
	addrStrList := []string{
		"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
		"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08",
		"vite_1630f8c0cf5eda3ce64bd49a0523b826f67b19a33bc2a5dcfb",
		"vite_1b1dfa00323aea69465366d839703547fec5359d6c795c8cef",
		"vite_27a258dd1ed0ce0de3f4abd019adacd1b4b163b879389d3eca",
		"vite_31a02e4f4b536e2d6d9bde23910cdffe72d3369ef6fe9b9239",
		"vite_383fedcbd5e3f52196a4e8a1392ed3ddc4d4360e4da9b8494e",
		"vite_41ba695ff63caafd5460dcf914387e95ca3a900f708ac91f06",
		"vite_545c8e4c74e7bb6911165e34cbfb83bc513bde3623b342d988",
		"vite_5a1b5ece654138d035bdd9873c1892fb5817548aac2072992e",
		"vite_70cfd586185e552635d11f398232344f97fc524fa15952006d",
		"vite_76df2a0560694933d764497e1b9b11f9ffa1524b170f55dda0",
		"vite_7b76ca2433c7ddb5a5fa315ca861e861d432b8b05232526767",
		"vite_7caaee1d51abad4047a58f629f3e8e591247250dad8525998a",
		"vite_826a1ab4c85062b239879544dc6b67e3b5ce32d0a1eba21461",
		"vite_89007189ad81c6ee5cdcdc2600a0f0b6846e0a1aa9a58e5410",
		"vite_9abcb7324b8d9029e4f9effe76f7336bfd28ed33cb5b877c8d",
		"vite_af60cf485b6cc2280a12faac6beccfef149597ea518696dcf3",
		"vite_c1090802f735dfc279a6c24aacff0e3e4c727934e547c24e5e",
		"vite_c10ae7a14649800b85a7eaaa8bd98c99388712412b41908cc0",
		"vite_d45ac37f6fcdb1c362a33abae4a7d324a028aa49aeea7e01cb",
		"vite_d8974670af8e1f3c4378d01d457be640c58644bc0fa87e3c30",
		"vite_e289d98f33c3ef5f1b41048c2cb8b389142f033d1df9383818",
		"vite_f53dcf7d40b582cd4b806d2579c6dd7b0b131b96c2b2ab5218",
		"vite_fac06662d84a7bea269265e78ea2d9151921ba2fae97595608",
	}

	for _, addrStr := range addrStrList {
		addr, _ := types.HexToAddress(addrStr)
		defaultBlockProducers = append(defaultBlockProducers, addr)
	}

	defaultSnapshotConsensusGroup := config.ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            1,
		PerCount:            3,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: config.ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}
	defaultCommonConsensusGroup := config.ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            3,
		PerCount:            1,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: config.ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}

	genesisConfig := &config.Genesis{
		GenesisAccountAddress: defaultGenesisAccountAddress,
		BlockProducers:        defaultBlockProducers,
	}

	if len(c.GenesisFile) > 0 {
		file, err := os.Open(c.GenesisFile)
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		genesisConfig = new(config.Genesis)
		if err := json.NewDecoder(file).Decode(genesisConfig); err != nil {
			log.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
	}

	if genesisConfig.SnapshotConsensusGroup == nil {
		genesisConfig.SnapshotConsensusGroup = &defaultSnapshotConsensusGroup
	}

	if genesisConfig.CommonConsensusGroup == nil {
		genesisConfig.CommonConsensusGroup = &defaultCommonConsensusGroup
	}

	// set fork points
	genesisConfig.ForkPoints = c.makeForkPointsConfig(genesisConfig)

	return genesisConfig
}

func (c *Config) makeChainConfig() *config.Chain {

	if len(c.KafkaProducers) == 0 {
		return &config.Chain{
			KafkaProducers: nil,
			OpenBlackBlock: c.OpenBlackBlock,
			LedgerGcRetain: c.LedgerGcRetain,
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
