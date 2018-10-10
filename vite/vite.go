package vite

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/miner"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite struct {
	config        *config.Config
	walletManager *wallet.Manager
	chain         chain.Chain
	verifier      consensus.Verifier
	miner         *miner.Miner
	Net           *net.Net
	p2p           *p2p.Server
}

func New(cfg *config.Config) (vite *Vite, err error) {
	// todo other module created and init

	net, err := net.New(&net.Config{
		Port:  uint16(cfg.FilePort),
		Chain: nil,
	})

	if err != nil {
		return
	}

	vite = &Vite{
		config: cfg,
		Net:    net,
	}

	//downloadLedger(cfg.IsDownload, cfg.DataDir)
	//
	//log := log15.New("module", "vite/backend")
	//vite := &Vite{config: cfg}
	//
	//vite.ledger = ledgerHandler.NewManager(vite, cfg.DataDir)
	//
	//vite.walletManager = wallet.New(cfg.DataDir)
	//vite.signer = signer.NewMaster(vite)
	//vite.signer.InitAndStartLoop()
	//
	//vite.pm = net.New(vite)
	//
	//vite.p2p = p2p.New(cfg.P2P)
	//
	//genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	//committee := consensus.NewCommittee(genesisTime, int32(cfg.MinerInterval), int32(len(consensus.DefaultMembers)))
	//vite.verifier = committee
	//
	//if cfg.Miner.Miner && cfg.Miner.Coinbase != "" {
	//	log.Info("Vite backend new: Start miner.")
	//	coinbase, _ := types.HexToAddress(cfg.Miner.Coinbase)
	//	vite.miner = miner.NewMiner(vite.ledger.Sc(), vite.ledger.RegisterFirstSyncDown, coinbase, committee)
	//	pwd := "123"
	//	vite.walletManager.KeystoreManager.Unlock(coinbase, pwd, 0)
	//	committee.Init()
	//	vite.miner.Init()
	//	vite.miner.Start()
	//	committee.Start()
	//}
	//vite.p2p.Start()
	//
	//fmt.Println("vite node start success you can find the runlog in", cfg.RunLogDir())

	return
}

func (v *Vite) Start(svr *p2p.Server) {
	v.p2p = svr
}

func (v *Vite) Protocols() []*p2p.Protocol {
	return v.Net.Protocols
}

func (v *Vite) Stop() {

}

func (v *Vite) Chain() chain.Chain {
	return v.chain
}
func (v *Vite) P2p() *p2p.Server {
	return v.p2p
}

func (v *Vite) WalletManager() *wallet.Manager {
	return v.walletManager
}

func (v *Vite) Miner() *miner.Miner {
	return v.miner
}
func (v *Vite) Verifier() consensus.Verifier {
	return v.verifier
}
