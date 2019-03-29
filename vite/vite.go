package vite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vitelabs/go-vite/vite/net/sbpn"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/wallet"
)

var (
	log = log15.New("module", "console/bridge")
)

type Vite struct {
	config *config.Config

	walletManager   *wallet.Manager
	accountVerifier verifier.Verifier
	chain           chain.Chain
	producer        producer.Producer
	net             net.Net
	pool            pool.BlockPool
	consensus       consensus.Consensus
	onRoad          *onroad.Manager
	p2p             p2p.Server
}

func New(cfg *config.Config, walletManager *wallet.Manager) (vite *Vite, err error) {
	// set fork points
	fork.SetForkPoints(cfg.ForkPoints)

	// chain
	chain := chain.NewChain(cfg.DataDir, cfg.Chain, cfg.Genesis)

	err = chain.Init()
	if err != nil {
		return nil, err
	}
	// pool
	pl, err := pool.NewPool(chain)
	if err != nil {
		return nil, err
	}
	// consensus
	cs := consensus.NewConsensus(chain)

	// sb verifier
	aVerifier := verifier.NewAccountVerifier(chain, cs)
	sbVerifier := verifier.NewSnapshotVerifier(chain, cs)

	verifier := verifier.NewVerifier(sbVerifier, aVerifier)
	// net
	net := net.New(&net.Config{
		Single:      cfg.Single,
		FileAddress: cfg.FileAddress,
		Chain:       chain,
		Verifier:    verifier,
	})

	// vite
	vite = &Vite{
		config:          cfg,
		walletManager:   walletManager,
		net:             net,
		chain:           chain,
		pool:            pl,
		consensus:       cs,
		accountVerifier: verifier,
	}

	// producer
	if cfg.Producer.Producer && cfg.Producer.Coinbase != "" {
		coinbase, index, err := parseCoinbase(cfg.Producer.Coinbase)
		//coinbase, err := types.HexToAddress(cfg.Producer.Coinbase)
		if err != nil {
			log.Error(fmt.Sprintf("coinBase parse fail. %v", cfg.Producer.Coinbase), "err", err)
			return nil, err
		}
		err = walletManager.MatchAddress(cfg.EntropyStorePath, *coinbase, index)

		if err != nil {
			log.Error(fmt.Sprintf("coinBase is not child of entropyStore, coinBase is : %v", cfg.Producer.Coinbase), "err", err)
			return nil, err
		}
		addressContext := &producer.AddressContext{
			EntryPath: cfg.EntropyStorePath,
			Address:   *coinbase,
			Index:     index,
		}
		vite.producer = producer.NewProducer(chain, net, addressContext, cs, sbVerifier, walletManager, pl)

		net.AddPlugin(sbpn.New(*coinbase, cs))
	}

	// onroad
	or := onroad.NewManager(net, pl, vite.producer, walletManager)

	// set onroad
	vite.onRoad = or
	return
}

func (v *Vite) Init() (err error) {
	vm.InitVmConfig(v.config.IsVmTest, v.config.IsUseVmTestParam, v.config.IsVmDebug, v.config.DataDir)

	//v.chain.Init()
	if v.producer != nil {
		if err := v.producer.Init(); err != nil {
			log.Error("Init producer failed, error is "+err.Error(), "method", "vite.Init")
			return err
		}
	}

	v.onRoad.Init(v.chain)

	return nil
}

func (v *Vite) Start(p2p p2p.Server) (err error) {
	v.p2p = p2p

	v.onRoad.Start()

	v.chain.Start()

	err = v.consensus.Init()
	if err != nil {
		return err
	}
	// hack
	v.pool.Init(v.net, v.walletManager, v.accountVerifier.GetSnapshotVerifier(), v.accountVerifier)

	v.consensus.Start()

	err = v.net.Start(p2p)
	if err != nil {
		return
	}

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

func (v *Vite) Net() net.Net {
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

func (v *Vite) Config() *config.Config {
	return v.config
}

func (v *Vite) P2P() p2p.Server {
	return v.p2p
}

func parseCoinbase(coinbaseCfg string) (*types.Address, uint32, error) {
	splits := strings.Split(coinbaseCfg, ":")
	if len(splits) != 2 {
		return nil, 0, errors.New("len is not equals 2.")
	}
	i, err := strconv.Atoi(splits[0])
	if err != nil {
		return nil, 0, err
	}
	addr, err := types.HexToAddress(splits[1])
	if err != nil {
		return nil, 0, err
	}

	return &addr, uint32(i), nil
}
