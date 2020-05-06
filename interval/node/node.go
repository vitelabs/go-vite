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

func (n *node) Init() {
	n.syncer.Init(n.ledger.Chain(), n.ledger.Pool())
	n.ledger.Init(n.syncer)
	n.consensus.Init()
	n.p2p.Init()
	if n.miner != nil {
		n.miner.Init()
	}
	n.wallet = wallet.NewWallet()
}

func (n *node) Start() {
	n.p2p.Start()
	n.ledger.Start()
	n.syncer.Start()
	n.consensus.Start()

	if n.miner != nil {
		n.miner.Start()
	}

	log.Info("node started...")
}

func (n *node) Stop() {
	close(n.closed)

	if n.miner != nil {
		n.miner.Stop()
	}
	n.consensus.Stop()
	n.syncer.Stop()
	n.ledger.Stop()
	n.p2p.Stop()
	n.wg.Wait()
	log.Info("node stopped...")
}

func (n *node) StartMiner() {
	if n.miner == nil {
		n.cfg.MinerCfg.HexCoinbase = n.wallet.CoinBase()
		n.miner = miner.NewMiner(n.ledger, n.syncer, n.bus, n.cfg.MinerCfg.CoinBase(), n.consensus)
		n.miner.Init()
	}
	n.miner.Start()
	log.Info("miner started...")
}

func (n *node) StopMiner() {
	if n.miner != nil {
		n.miner.Stop()
		log.Info("miner stopped...")
	}
}

func (n *node) Leger() ledger.Ledger {
	return n.ledger
}

func (n *node) Wallet() wallet.Wallet {
	return n.wallet
}
func (n *node) P2P() p2p.P2P {
	return n.p2p
}
