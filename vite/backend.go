package vite

import (
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/wallet"

	"github.com/vitelabs/go-vite/ledger/handler_interface"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/miner"
	"github.com/vitelabs/go-vite/signer"
	"github.com/vitelabs/go-vite/vitedb"
	"log"
	"time"
)

type Vite struct {
	config        *Config
	ledger        *ledgerHandler.Manager
	p2p           *p2p.Server
	pm            *protocols.ProtocolManager
	walletManager *wallet.Manager
	signer        *signer.Master
	verifier      consensus.Verifier
	miner         *miner.Miner
}

var (
	defaultP2pConfig = &p2p.Config{}
	DefaultConfig    = &Config{
		DataDir:   common.DefaultDataDir(),
		P2pConfig: defaultP2pConfig,
		Miner:     false,
		Coinbase:  "",
	}
)

type Config struct {
	DataDir   string
	P2pConfig *p2p.Config
	Miner     bool
	Coinbase  string
	MinerInterval int
}

func New(cfg *Config) (*Vite, error) {
	//viteconfig.LoadConfig("gvite")
	//fmt.Printf("%+v\n", config.Map())

	if cfg == nil {
		cfg = DefaultConfig
	}

	if cfg.P2pConfig == nil {
		cfg.P2pConfig = defaultP2pConfig
	}

	vitedb.InitDataBaseEnv(cfg.DataDir)
	vite := &Vite{config: cfg}

	vite.ledger = ledgerHandler.NewManager(vite)

	vite.walletManager = wallet.NewManagerAndInit(cfg.DataDir)
	vite.signer = signer.NewMaster(vite)
	vite.signer.InitAndStartLoop()

	vite.pm = protocols.NewProtocolManager(vite)

	var initP2pErr error
	vite.p2p, initP2pErr = p2p.NewServer(cfg.P2pConfig, vite.pm.HandlePeer)
	if initP2pErr != nil {
		log.Fatal(initP2pErr)
	}

	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, 6, int32(len(consensus.DefaultMembers)))
	vite.verifier = committee

	if cfg.Miner && cfg.Coinbase != "" {
		coinbase, _ := types.HexToAddress(cfg.Coinbase)
		vite.miner = miner.NewMiner(vite.ledger.Sc(), coinbase, committee)
		pwd := "123"
		vite.walletManager.KeystoreManager.Unlock(coinbase, pwd, time.Second*10)
		committee.Init()
		vite.miner.Init()
		vite.miner.Start()
		committee.Start()
	}
	vite.p2p.Start()
	return vite, nil
}

func (v *Vite) Ledger() handler_interface.Manager {
	return v.ledger
}

func (v *Vite) P2p() *p2p.Server {
	return v.p2p
}

func (v *Vite) Pm() protoInterface.ProtocolManager {
	return v.pm
}

func (v *Vite) WalletManager() *wallet.Manager {
	return v.walletManager
}

func (v *Vite) Signer() *signer.Master {
	return v.signer
}

func (v *Vite) Config() *Config {
	return v.config
}

func (v *Vite) Miner() *miner.Miner {
	return v.miner
}
func (v *Vite) Verifier() consensus.Verifier {
	return v.verifier
}
