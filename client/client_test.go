package client

import (
	"fmt"
	"math/big"
	"os/user"
	"path"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/entropystore"
)

var WalletDir string

func init() {
	current, _ := user.Current()
	home := current.HomeDir
	WalletDir = path.Join(home, "Library/GVite/devdata/wallet")
}

var Wallet2 *entropystore.Manager

func PreTest() error {
	w := wallet.New(&wallet.Config{
		DataDir:        WalletDir,
		MaxSearchIndex: 100000,
	})
	w.Start()

	w2, err := w.RecoverEntropyStoreFromMnemonic("extend excess vibrant crop split vehicle order veteran then fog panel appear frozen deer logic path yard tenant bag nuclear witness annual silent fold", "123456")
	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		return err
	}
	err = w2.Unlock("123456")
	if err != nil {

		fmt.Errorf("wallet error, %+v", err)
		return err
	}

	Wallet2 = w2
	return nil
}

func TestWallet(t *testing.T) {
	err := PreTest()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	/**
	vite_165a295e214421ef1276e79990533953e901291d29b2d4851f
	vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda
	vite_e7e7fd6c532d38d0a8c1158ea89d41f7ffddeef3b6e11309b9
	vite_fcdb46a9ce7c4fd1d3321636e707660255d70e39062cd16460
	vite_78290408d7ec2293e2315cb7d98260629ede76882333e77161
	vite_3dfa8bd841bc4ed351953a12f4edbd8364ed388908136f50a1
	vite_73c6b08e401608bca17272e7c59508f2e549c221ae7efccd53
	vite_9d2dafb40aec2d287fa660e879a65b108cae355bf85e6d9ae6
	vite_260033138517d251cfa3a907e5a4f0c673d656909108e1c832
	vite_8d5de7117bbf8c1fb911ba68759b5c34dea7f63987771662ba
	*/
	t.Log("----------------------Wallet2----------------------")
	for i := uint32(0); i < 10; i++ {
		_, key, err := Wallet2.DeriveForIndexPath(i)
		if err != nil {
			t.Error(err)
			return
		}
		addr, _ := key.Address()
		fmt.Println(addr)
	}
	t.Log("----------------------Wallet2----------------------")
}

func TestClient_SubmitRequestTx(t *testing.T) {
	if err := PreTest(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}
	self, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
	if err != nil {
		t.Error(err)
		return
	}
	to, err := types.HexToAddress("vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda")
	if err != nil {
		t.Error(err)
		return
	}

	block, err := client.BuildNormalRequestBlock(RequestTxParams{
		ToAddr:   to,
		SelfAddr: self,
		Amount:   big.NewInt(10000),
		TokenId:  ledger.ViteTokenId,
		Data:     []byte("hello pow"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}

	err = rpc.SendRawTx(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("submit request tx success.", block.Hash, block.Height)
}

func TestClient_CreateContract(t *testing.T) {
	if err := PreTest(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}
	self, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
	if err != nil {
		t.Error(err)
		return
	}
	definition := ``
	code := ``
	block, err := client.BuildRequestCreateContractBlock(RequestCreateContractParams{
		SelfAddr: self,
		abiStr:   definition,
		metaParams: api.CreateContractDataParam{
			Gid:         types.DELEGATE_GID,
			ConfirmTime: 12,
			SeedCount:   12,
			QuotaRatio:  10,
			HexCode:     code,
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}

	err = rpc.SendRawTx(block)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("submit request tx success.", block.Hash, block.Height)
}

func TestClient_SubmitResponseTx(t *testing.T) {
	PreTest()
	to, err := types.HexToAddress("vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(to)
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	requestHash := types.HexToHashPanic("1058ac419ffa5f8cfa8bf3a19e8f4cf870ec0956025dc4ebc17793344fd2e67e")
	t.Log("receive request.", requestHash)
	block, err := client.BuildResponseBlock(ResponseTxParams{
		SelfAddr:    to,
		RequestHash: requestHash,
	}, nil)

	if err != nil {
		t.Fatal(err)
	}

	t.Log("receive request.", requestHash, block.Amount)

	err = client.SignData(Wallet2, block)
	if err != nil {
		t.Fatal(err)
	}
	err = rpc.SendRawTx(block)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_QueryOnroad(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := types.HexToAddress("vite_2ca3c5f1f18b38f865eb47196027ae0c50d0c21e67774abdda")
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := rpc.GetOnroadBlocksByAddress(addr, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) > 0 {
		for _, v := range blocks {
			t.Log(v.Height, v.AccountAddress, v.ToAddress, *v.Amount, v.Hash)
		}
	}
}

func TestClient_GetBalanceAll(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	addr, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(err)
		return
	}

	balance, onroad, err := client.GetBalanceAll(addr)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range balance.TokenBalanceInfoMap {
		t.Log(k, "balance", v.TokenInfo.TokenSymbol, v.TotalAmount)
	}
	for k, v := range onroad.TokenBalanceInfoMap {
		t.Log(k, "onroad", v.TokenInfo.TokenSymbol, v.TotalAmount)
	}
}

func TestClient_GetBalance(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	client, e := NewClient(rpc)
	if e != nil {
		t.Error(e)
		return
	}

	addr, err := types.HexToAddress("vite_1b351d987dd194ea7f8146a45e7b2625c1d9d483505fc524e8")
	if err != nil {
		t.Error(err)
		return
	}

	balance, onroad, err := client.GetBalance(addr, ledger.ViteTokenId)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("balance", balance.String())
	t.Log("onroad", onroad.String())
}
