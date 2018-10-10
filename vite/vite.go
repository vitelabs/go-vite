package vite

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite struct {
	config *config.Config

	walletManager *wallet.Manager
	chain         chain.Chain
	producer      producer.Producer
	net           *net.Net
	pool          pool.BlockPool
	consensus     consensus.Consensus
	onRoad        *onroad.Manager
}

func New(cfg *config.Config, walletManager *wallet.Manager) (vite *Vite, err error) {
	// todo other module created and init

	// chain
	chain := chain.NewChain(cfg)

	// pool
	pl := pool.NewPool(chain)
	genesis := chain.GetGenesisSnapshotBlock()

	// consensus
	cs := consensus.NewConsensus(*genesis.Timestamp, chain)

	// net
	// TODO verifier.NewAccountVerifier
	aVerifier := verifier.NewAccountVerifier(chain, cs)
	net, err := net.New(&net.Config{
		Port:     uint16(cfg.FilePort),
		Chain:    chain,
		Verifier: aVerifier,
	})
	if err != nil {
		log.Error("net.New failed. Error is "+err.Error(), "method", "new Vite()")
		return nil, err
	}

	// vite
	vite = &Vite{
		config:    cfg,
		net:       net,
		chain:     chain,
		pool:      pl,
		consensus: cs,
	}
	// onroad
	or := onroad.NewManager(vite.net, vite.chain, vite.pool, vite.producer, vite.walletManager)
	// set onroad
	vite.onRoad = or

	// producer
	if cfg.Producer.Producer && cfg.Producer.Coinbase != "" {
		coinbase, err := types.HexToAddress(cfg.Producer.Coinbase)

		if err != nil {
			log.Error("Coinbase parse failed from config file. Error is "+err.Error(), "method", "new Vite()")
			return nil, err
		}

		sbVerifier := verifier.NewSnapshotVerifier(chain, cs)
		vite.producer = producer.NewProducer(chain, net, coinbase, cs, sbVerifier, walletManager, pl)
	}
	return
}

func (v *Vite) Init() (err error) {
	v.chain.Init()
	if err := v.producer.Init(); err != nil {
		log.Error("Init producer failed, error is "+err.Error(), "method", "vite.Init")
		return err
	}
	return nil
}

func (v *Vite) Start() (err error) {
	v.chain.Start()
	if v.producer != nil {
		err := v.producer.Start()
		if err != nil {
			log.Error("producer.Start failed, error is "+err.Error(), "method", "vite.Start")
			return err
		}
	}
	return nil
}

func (v *Vite) Stop() (err error) {
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
