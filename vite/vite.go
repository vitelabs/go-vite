package vite

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite struct {
	config *config.Config

	walletManager    *wallet.Manager
	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier
	chain            chain.Chain
	producer         producer.Producer
	net              *net.Net
	pool             pool.BlockPool
	consensus        consensus.Consensus
	onRoad           *onroad.Manager
}

func New(cfg *config.Config, walletManager *wallet.Manager) (vite *Vite, err error) {

	// chain
	chain := chain.NewChain(cfg)

	// pool
	pl := pool.NewPool(chain)
	genesis := chain.GetGenesisSnapshotBlock()

	// consensus
	cs := consensus.NewConsensus(*genesis.Timestamp, chain)

	// net
	aVerifier := verifier.NewAccountVerifier(chain, cs)
	net, err := net.New(&net.Config{
		Single:   cfg.Single,
		Port:     uint16(cfg.FilePort),
		Chain:    chain,
		Verifier: aVerifier,
		Topology: cfg.Topology,
		Topic:    cfg.Topic,
		Interval: cfg.Interval,
	})
	if err != nil {
		log.Error("net.New failed. Error is "+err.Error(), "method", "new Vite()")
		return nil, err
	}

	// sb verifier
	sbVerifier := verifier.NewSnapshotVerifier(chain, cs)

	// vite
	vite = &Vite{
		config:           cfg,
		walletManager:    walletManager,
		net:              net,
		chain:            chain,
		pool:             pl,
		consensus:        cs,
		snapshotVerifier: sbVerifier,
		accountVerifier:  aVerifier,
	}

	// onroad
	or := onroad.NewManager(vite.net, vite.pool, vite.producer, vite.walletManager)

	// set onroad
	vite.onRoad = or

	// producer
	if cfg.Producer.Producer && cfg.Producer.Coinbase != "" {
		coinbase, err := types.HexToAddress(cfg.Producer.Coinbase)

		if err != nil {
			log.Error("Coinbase parse failed from config file. Error is "+err.Error(), "method", "new Vite()")
			return nil, err
		}

		vite.producer = producer.NewProducer(chain, net, coinbase, cs, sbVerifier, walletManager, pl)
	}
	return
}

func (v *Vite) Init() (err error) {
	vm.InitVmConfig(v.config.IsVmTest)

	v.chain.Init()
	if v.producer != nil {
		if err := v.producer.Init(); err != nil {
			log.Error("Init producer failed, error is "+err.Error(), "method", "vite.Init")
			return err
		}
	}

	v.onRoad.Init(v.chain)

	return nil
}

func (v *Vite) Start(p2p *p2p.Server) (err error) {
	v.onRoad.Start()

	v.chain.Start()

	err = v.consensus.Init()
	if err != nil {
		return err
	}
	// hack
	v.pool.Init(v.net, v.walletManager, v.snapshotVerifier, v.accountVerifier)

	v.consensus.Start()
	v.net.Start(p2p)
	v.pool.Start()
	if v.producer != nil {

		if err := v.producer.Start(); err != nil {
			log.Error("producer.Start failed, error is "+err.Error(), "method", "vite.Start")
			return err
		}
	}
	return nil
}

func (v *Vite) Stop() (err error) {

	v.net.Stop()
	v.pool.Stop()

	if v.producer != nil {
		if err := v.producer.Stop(); err != nil {
			log.Error("producer.Stop failed, error is "+err.Error(), "method", "vite.Stop")
			return err
		}
	}
	v.consensus.Stop()
	v.chain.Stop()
	v.onRoad.Stop()
	return nil
}

func (v *Vite) Chain() chain.Chain {
	return v.chain
}

func (v *Vite) Net() *net.Net {
	return v.net
}

func (v *Vite) WalletManager() *wallet.Manager {
	return v.walletManager
}

func (v *Vite) Producer() producer.Producer {
	return v.producer
}

func (v *Vite) Pool() pool.BlockPool {
	return v.pool
}

func (v *Vite) Consensus() consensus.Consensus {
	return v.consensus
}

func (v *Vite) OnRoad() *onroad.Manager {
	return v.onRoad
}
