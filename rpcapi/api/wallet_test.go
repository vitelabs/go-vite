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
	addr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")

	password := "123456"

	unlockAddr(w, password, genesisAccountPrivKeyStr)

	vite, err := startVite(w, password, t)

	waApi := NewWalletApi(vite)
	onRoadApi := NewPrivateOnroadApi(vite.OnRoad())

	//l := NewLedgerApi(vite)
	t.Log(waApi.Status())

	vite.OnRoad().StartAutoReceiveWorker(addr, nil)
	for _, v := range vite.OnRoad().ListWorkingAutoReceiveWorker() {
		wLog.Info(v.String())
	}

	waitOnroad(onRoadApi, addr, t)
	printBalance(vite, addr)

	printQuota(vite, addr)

	waitOnroad(onRoadApi, contracts.AddressPledge, t)

	byt, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr)

	parms := CreateTransferTxParms{
		SelfAddr:    addr,
		ToAddr:      contracts.AddressPledge,
		TokenTypeId: ledger.ViteTokenId,
		Passphrase:  password,
		Amount:      new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)).String(),
		Data:        byt,
	}
	balance := printBalance(vite, addr)

	contractPrevHeight := printHeight(vite, contracts.AddressPledge)

	if balance.Sign() == 0 {
		err = waApi.CreateTxWithPassphrase(parms)
		if err != nil {
			t.Error(err)
			return
		}
	}

	// wait address height ++
	for {
		if printHeight(vite, contracts.AddressPledge) > contractPrevHeight {
			break
		}
		time.Sleep(time.Second)
	}

	// wait snapshot ++
	prevHeight := printSnapshot(vite)
	for {
		if printSnapshot(vite) > prevHeight {
			break
		}
		time.Sleep(time.Second)
	}
	printQuota(vite, addr)
}
func startVite(w *wallet.Manager, password string, t *testing.T) (*vite.Vite, error) {
	coinbase := unlockAddr(w, password, accountPrivKeyStr)

	config := &config.Config{
		DataDir: common.DefaultDataDir(),
		Producer: &config.Producer{
			Producer: true,
			Coinbase: coinbase.String(),
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
	err = vite.Start()
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
func printQuota(vite *vite.Vite, addr types.Address) {
	head := vite.Chain().GetLatestSnapshotBlock()
	amount := vite.Chain().GetPledgeAmount(head.Hash, addr)
	wLog.Info("print quota", "quota", amount.String(), "snapshotHash", head.Hash, "snapshotHeight", head.Height)
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
	info, e := api.GetAccountOnroadInfo(addr)
	if e != nil {
		panic(e)
		return
	}

	total := big.NewInt(0)
	total.SetString(info.TotalNumber, 10)
	for total.Sign() > 0 {
		wLog.Info("print onroad size", "size", total.String(), "addr", addr.String())
		time.Sleep(time.Second)
	}
}
