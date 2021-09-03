package vite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vitelabs/go-vite/common/config"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/common/upgrade"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
	"github.com/vitelabs/go-vite/ledger/onroad"
	"github.com/vitelabs/go-vite/ledger/pool"
	"github.com/vitelabs/go-vite/ledger/verifier"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/wallet"
)

var (
	log = log15.New("module", "console/bridge")
)

type Vite struct {
	config *config.Config

	walletManager *wallet.Manager
	verifier      verifier.Verifier
	chain         chain.Chain
	producer      producer.Producer
	net           net.Net
	pool          pool.BlockPool
	consensus     consensus.Consensus
	onRoad        *onroad.Manager
}

func New(cfg *config.Config, walletManager *wallet.Manager) (vite *Vite, err error) {
	// set upgrade
	upgrade.InitUpgradeBox(cfg.UpgradeCfg.MakeUpgradeBox())

	var account *wallet.Account
	if cfg.Producer.IsMine() {
		account, err = walletManager.AccountAtIndex(cfg.EntropyStorePath, cfg.Producer.GetCoinbase(), cfg.Producer.GetIndex())
		if err != nil {
			log.Error(fmt.Sprintf("coinBase is not child of entropyStore, coinBase is : %v", cfg.Producer.Coinbase), "err", err)
			return nil, err
		}

		cfg.Net.MineKey, err = account.PrivateKey()
		if err != nil {
			return
		}
	}

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
	cs := consensus.NewConsensus(chain, pl)

	verifier := verifier.NewVerifier2(chain, cs)

	// net
	net, err := net.New(cfg.Net, chain, verifier, cs, pl)
	if err != nil {
		return
	}

	// vite
	vite = &Vite{
		config:        cfg,
		walletManager: walletManager,
		net:           net,
		chain:         chain,
		pool:          pl,
		consensus:     cs,
		verifier:      verifier,
	}

	if account != nil {
		vite.producer = producer.NewProducer(chain, net, account, cs, verifier.GetSnapshotVerifier(), pl)
	}
	// set onroad
	vite.onRoad = onroad.NewManager(net, pl, vite.producer, vite.consensus, account)
	return
}

func (v *Vite) Init() (err error) {
	vm.InitVMConfig(v.config.IsVmTest, v.config.IsUseVmTestParam, v.config.IsUseQuotaTestParam, v.config.IsVmDebug, v.config.DataDir)

	//v.chain.Init()
	if v.producer != nil {
		if err := v.producer.Init(); err != nil {
			log.Error("Init producer failed, error is "+err.Error(), "method", "vite.Init")
			return err
		}
	}

	// initOnRoadPool
	v.onRoad.Init(v.chain)
	v.verifier.InitOnRoadPool(v.onRoad)

	return nil
}

func (v *Vite) Start() (err error) {
	v.onRoad.Start()

	v.chain.Start()

	err = v.consensus.Init(nil)
	if err != nil {
		return err
	}
	// hack
	v.pool.Init(v.net, v.verifier.GetSnapshotVerifier(), v.verifier, v.consensus)

	v.consensus.Start()

	err = v.net.Start()
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

func (v *Vite) Verifier() verifier.Verifier {
	return v.verifier
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
