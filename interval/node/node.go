package node

import (
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/interval/chain"
	"github.com/vitelabs/go-vite/interval/common/config"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/consensus"
	"github.com/vitelabs/go-vite/interval/ledger"
	"github.com/vitelabs/go-vite/interval/miner"
	"github.com/vitelabs/go-vite/interval/p2p"
	"github.com/vitelabs/go-vite/interval/syncer"
	"github.com/vitelabs/go-vite/interval/wallet"
)

type Node interface {
	Init()
	Start()
	Stop()
	StartMiner()
	StopMiner()
	Leger() ledger.Ledger
	P2P() p2p.P2P
	Wallet() wallet.Wallet
}

func NewNode(cfg *config.Node) (Node, error) {
	e := cfg.Check()
	if e != nil {
		return nil, e
	}
	self := &node{}
	self.bus = EventBus.New()
	self.closed = make(chan struct{})
	self.cfg = cfg
	self.p2p = p2p.NewP2P(self.cfg.P2pCfg)
	self.syncer = syncer.NewSyncer(self.p2p, self.bus)
	self.bc = chain.NewChain(self.cfg.ChainCfg)
	self.ledger = ledger.NewLedger(self.bc)
	self.consensus = consensus.NewConsensus(chain.GetGenesisSnapshot().Timestamp(), self.cfg.ConsensusCfg)

	if self.cfg.MinerCfg.Enabled {
		self.miner = miner.NewMiner(self.ledger, self.syncer, self.bus, self.cfg.MinerCfg.CoinBase(), self.consensus)
	}
	return self, nil
}

type node struct {
	bc        chain.BlockChain
	p2p       p2p.P2P
	syncer    syncer.Syncer
	ledger    ledger.Ledger
	consensus consensus.Consensus
	miner     miner.Miner
	wallet    wallet.Wallet
	bus       EventBus.Bus

	cfg    *config.Node
	closed chan struct{}
	wg     sync.WaitGroup
}

func (self *node) Init() {
	self.syncer.Init(self.ledger.Chain(), self.ledger.Pool())
	self.ledger.Init(self.syncer)
	self.consensus.Init()
	self.p2p.Init()
	if self.miner != nil {
		self.miner.Init()
	}
	self.wallet = wallet.NewWallet()
}

func (self *node) Start() {
	self.p2p.Start()
	self.ledger.Start()
	self.syncer.Start()
	self.consensus.Start()

	if self.miner != nil {
		self.miner.Start()
	}

	log.Info("node started...")
}

func (self *node) Stop() {
	close(self.closed)

	if self.miner != nil {
		self.miner.Stop()
	}
	self.consensus.Stop()
	self.syncer.Stop()
	self.ledger.Stop()
	self.p2p.Stop()
	self.wg.Wait()
	log.Info("node stopped...")
}

func (self *node) StartMiner() {
	if self.miner == nil {
		self.cfg.MinerCfg.HexCoinbase = self.wallet.CoinBase()
		self.miner = miner.NewMiner(self.ledger, self.syncer, self.bus, self.cfg.MinerCfg.CoinBase(), self.consensus)
		self.miner.Init()
	}
	self.miner.Start()
	log.Info("miner started...")
}

func (self *node) StopMiner() {
	if self.miner != nil {
		self.miner.Stop()
		log.Info("miner stopped...")
	}
}

func (self *node) Leger() ledger.Ledger {
	return self.ledger
}

func (self *node) Wallet() wallet.Wallet {
	return self.wallet
}
func (self *node) P2P() p2p.P2P {
	return self.p2p
}
