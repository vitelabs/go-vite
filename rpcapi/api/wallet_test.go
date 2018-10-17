package api

import (
	"flag"
	"fmt"
	"testing"

	"time"

	"math/big"

	"strconv"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/wallet"
)

var wLog = log15.New("module", "walletTest")

var genesisAccountPrivKeyStr string
var accountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "g", "", "")
	flag.StringVar(&accountPrivKeyStr, "p", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)
}

func TestParse(t *testing.T) {
}

func TestWallet(t *testing.T) {
	w := wallet.New(nil)
	password := "123456"

	unlockAll(w)
	genesisAddr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")

	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")
	vite, err := startVite(w, &addr, t)

	t1, _ := time.Parse(time.RFC3339, "2018-10-12T16:19:28+08:00")

	vite.Consensus().ReadByTime(types.SNAPSHOT_GID, t1)

	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite)

	//l := NewLedgerApi(vite)
	t.Log(waApi.Status())

	vite.OnRoad().StartAutoReceiveWorker(genesisAddr, nil)
	for _, v := range vite.OnRoad().ListWorkingAutoReceiveWorker() {
		wLog.Info(v.String())
	}
	// if has no balance
	if printBalance(vite, genesisAddr).Sign() == 0 {
		waitOnroad(onRoadApi, genesisAddr, t)
		//time.Sleep(time.Minute)
	}

	// if has no quota
	if printQuota(vite, genesisAddr).Sign() == 0 {
		if printPledge(vite, genesisAddr, t).Sign() == 0 {
			waitContractOnroad(onRoadApi, contracts.AddressPledge, t)

			if printPledge(vite, genesisAddr, t).Sign() == 0 {
				// wait snapshot ++
				waitSnapshotInc(vite, t)

				byt, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, genesisAddr)
				parms := CreateTransferTxParms{
					SelfAddr:    genesisAddr,
					ToAddr:      contracts.AddressPledge,
					TokenTypeId: ledger.ViteTokenId,
					Passphrase:  password,
					Amount:      new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)).String(),
					Data:        byt,
					Difficulty:  new(big.Int).SetUint64(pow.FullThreshold),
				}
				err = waApi.CreateTxWithPassphrase(parms)
				if err != nil {
					t.Error(err)
					return
				}
			}
			waitContractOnroad(onRoadApi, contracts.AddressPledge, t)
		}
		waitQuota(vite, genesisAddr)
	}
}

func waitQuota(vite *vite.Vite, genesisAddr types.Address) {
	for {
		quota := printQuota(vite, genesisAddr)
		if quota.Sign() > 0 {
			break
		}
		printSnapshot(vite)
		printHeight(vite, contracts.AddressPledge)
	}
}
func printPledge(vite *vite.Vite, addr types.Address, t *testing.T) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	wLog.Info("print pledge", "height", strconv.FormatUint(head.Height, 10), "pledge", amount.String(), "addr", addr.String())
	return amount
}
func startVite(w *wallet.Manager, coinbase *types.Address, t *testing.T) (*vite.Vite, error) {
	//p2pServer, err := p2p.New(&p2p.Config{
	//	BootNodes: []string{
	//		"vnode://6d72c01e467e5280acf1b63f87afd5b6dcf8a596d849ddfc9ca70aab08f10191@192.168.31.146:8483",
	//		"vnode://1ceabc6c2b751b352a6d719b4987f828bb1cf51baafa4efac38bc525ed61059d@192.168.31.190:8483",
	//		"vnode://8343b3f2bc4e8e521d460cadab3e9f1e61ba57529b3fb48c5c076845c92e75d2@192.168.31.193:8483",
	//	},
	//	DataDir: path.Join(common.DefaultDataDir(), "/p2p"),
	//})

	wLog.Info(coinbase.String(), "coinbase")
	config := &config.Config{
		DataDir: common.DefaultDataDir(),
		Producer: &config.Producer{
			Producer: true,
			Coinbase: coinbase.String(),
		},
		Vm: &config.Vm{IsVmTest: false},
		Net: &config.Net{
			Single: true,
		},
	}
	vite, err := vite.New(config, w)
	if err != nil {
		t.Error(err)
		return nil, err
	}

	//p2pServer.Protocols = append(p2pServer.Protocols, vite.Net().Protocols()...)

	err = vite.Init()
	if err != nil {
		t.Error(err)
		return nil, err
	}

	//err = vite.Start(p2pServer)
	err = vite.Start(nil)
	if err != nil {
		t.Error(err)
		return nil, err
	}
	//err = p2pServer.Start()
	if err != nil {
		t.Error(err)
		return nil, err
	}
	return vite, nil
}
func printHeight(vite *vite.Vite, addr types.Address) uint64 {
	height, e := vite.Chain().GetLatestAccountBlock(&addr)
	if height == nil {
		wLog.Info("print height", "height", "0", "addr", addr.String(), "err", e)
		return 0
	} else {
		wLog.Info("print height", "height", strconv.FormatUint(height.Height, 10), "addr", addr.String(), "err", e)
	}
	return height.Height

}
func printQuota(vite *vite.Vite, addr types.Address) *big.Int {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	wLog.Info("print quota", "quota", amount.String(), "snapshotHash", head.Hash, "snapshotHeight", head.Height)
	return amount
}
func printSnapshot(vite *vite.Vite) uint64 {
	block := vite.Chain().GetLatestSnapshotBlock()
	wLog.Info("print snapshot", "height", strconv.FormatUint(block.Height, 10), "hash", block.Hash)
	return block.Height
}
func printBalance(vite *vite.Vite, addr types.Address) *big.Int {
	balance, _ := vite.Chain().GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
	//t.Log(balance)
	wLog.Info("print balance", "balance", balance.String(), "addr", addr.String())
	return balance
}

func unlockAddr(w *wallet.Manager, passwd string, priKey string) types.Address {
	w.KeystoreManager.ImportPriv(priKey, passwd)
	accountPrivKey, _ := ed25519.HexToPrivateKey(priKey)
	accountPubKey := accountPrivKey.PubByte()
	addr := types.PubkeyToAddress(accountPubKey)

	w.KeystoreManager.Lock(addr)
	err := w.KeystoreManager.Unlock(addr, passwd, 0)
	wLog.Info("unlock address", "address", addr.String(), "r", err)
	return addr
}

func waitOnroad(api *PrivateOnroadApi, addr types.Address, t *testing.T) {
	for {
		info, e := api.GetAccountOnroadInfo(addr)
		if e != nil {
			panic(e)
			return
		}

		if info == nil {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		total := big.NewInt(0)
		total.SetString(info.TotalNumber, 10)
		for total.Sign() == 0 {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		wLog.Info("print onroad size", "size", total.String(), "addr", addr.String())
		time.Sleep(time.Second)
	}
}

func waitContractOnroad(api *PrivateOnroadApi, addr types.Address, t *testing.T) {
	for {
		info, e := api.GetOnroadBlocksByAddress(addr, 0, 1000)
		if e != nil {
			panic(e)
			return
		}

		if len(info) == 0 {
			wLog.Info("print onroad size", "size", 0, "addr", addr.String())
			return
		}
		wLog.Info("print onroad size", "size", len(info), "addr", addr.String())
		time.Sleep(time.Second)
	}
}

func waitSnapshotInc(vite *vite.Vite, t *testing.T) {
	// wait snapshot ++
	prevHeight := printSnapshot(vite)
	for {
		if printSnapshot(vite) > prevHeight {
			break
		}
		time.Sleep(time.Second)
	}
}

func onroadNum(api *PrivateOnroadApi, addr types.Address, t *testing.T) int {
	info, e := api.GetAccountOnroadInfo(addr)
	if e != nil {
		panic(e)
		return 0
	}

	if info == nil {
		wLog.Info("print onroadNum size", "size", 0, "addr", addr.String())
		return 0
	}
	total := big.NewInt(0)
	total.SetString(info.TotalNumber, 10)

	wLog.Info("print onroadNum size", "size", total.String(), "addr", addr.String())
	return int(total.Int64())
}

func contractOnroadNum(api *PrivateOnroadApi, addr types.Address, t *testing.T) int {
	info, e := api.GetOnroadBlocksByAddress(addr, 0, 1000)
	if e != nil {
		panic(e)
		return 0
	}
	if len(info) == 0 {
		wLog.Info("print contractOnroadNum size", "size", 0, "addr", addr.String())
		return 0
	}
	wLog.Info("print contractOnroadNum size", "size", len(info), "addr", addr.String())
	return len(info)
}

func TestWalletBalance(t *testing.T) {
	w := wallet.New(nil)

	unlockAll(w)

	addr, _ := types.HexToAddress("vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1")

	vite, err := startVite(w, &addr, t)
	if err != nil {
		panic(err)
	}

	printBalance(vite, addr)

	waitSnapshotInc(vite, t)
}

var password = "123456"

func unlockAll(w *wallet.Manager) []types.Address {
	results := w.KeystoreManager.Addresses()

	for _, r := range results {
		err := w.KeystoreManager.Unlock(r, password, 0)
		if err != nil {
			log.Error("unlock fail.", "err", err, "address", r.String())
		}
	}
	return results
}

type CreateReceiveTxParms struct {
	SelfAddr   types.Address
	FromHash   types.Hash
	PrivKeyStr string
}

func ReceiveOnroadTx(vite *vite.Vite, params CreateReceiveTxParms) error {
	chain := vite.Chain()
	pool := vite.Pool()

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: params.SelfAddr,
		FromBlockHash:  &params.FromHash,
	}
	privKey, _ := ed25519.HexToPrivateKey(params.PrivKeyStr)
	pubKey := privKey.PubByte()

	g, e := generator.NewGenerator(chain, nil, nil, &params.SelfAddr)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(privKey, data), pubKey, nil
	})
	if e != nil {
		newerr, _ := TryMakeConcernedError(e)
		return newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		return pool.AddDirectAccountBlock(params.SelfAddr, result.BlockGenList[0])
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}
