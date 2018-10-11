package api

import (
	"flag"
	"fmt"
	"testing"

	"time"

	"math/big"

	"strconv"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
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

func TestWallet(t *testing.T) {
	w := wallet.New(nil)
	password := "123456"

	genesisAddr := unlockAddr(w, password, genesisAccountPrivKeyStr)

	vite, err := startVite(w, password, t)

	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite.OnRoad())

	//l := NewLedgerApi(vite)
	t.Log(waApi.Status())

	time.Sleep(2 * time.Second)
	vite.OnRoad().StartAutoReceiveWorker(genesisAddr, nil)
	for _, v := range vite.OnRoad().ListWorkingAutoReceiveWorker() {
		wLog.Info(v.String())
	}

	waitOnroad(onRoadApi, genesisAddr, t)
	//time.Sleep(time.Minute)

	printBalance(vite, genesisAddr)

	printQuota(vite, genesisAddr)

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
	printBalance(vite, genesisAddr)

	printHeight(vite, contracts.AddressPledge)

	if contractOnroadNum(onRoadApi, contracts.AddressPledge, t) == 0 {
		// wait snapshot ++
		prevHeight := printSnapshot(vite)
		for {
			if printSnapshot(vite) > prevHeight {
				break
			}
			time.Sleep(time.Second)
		}
		err = waApi.CreateTxWithPassphrase(parms)
		if err != nil {
			t.Error(err)
			return
		}
	}

	waitContractOnroad(onRoadApi, contracts.AddressPledge, t)

	// wait snapshot ++
	prevHeight := printSnapshot(vite)
	for {
		if printSnapshot(vite) > prevHeight {
			break
		}
		time.Sleep(time.Second)
	}
	for {
		quota := printQuota(vite, genesisAddr)
		if quota.Sign() > 0 {
			break
		}
		printSnapshot(vite)
		printHeight(vite, contracts.AddressPledge)
	}
	printQuota(vite, genesisAddr)
}
func startVite(w *wallet.Manager, password string, t *testing.T) (*vite.Vite, error) {
	coinbase := unlockAddr(w, password, accountPrivKeyStr)

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
	err = vite.Init()
	if err != nil {
		t.Error(err)
		return nil, err
	}
	err = vite.Start(nil)
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
