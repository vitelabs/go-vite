package client

import (
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path"
	"testing"

	"github.com/vitelabs/go-vite/wallet/entropystore"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/wallet"
)

var WalletDir string

func init() {
	current, _ := user.Current()
	home := current.HomeDir
	WalletDir = path.Join(home, "Library/GVite/devdata/wallet")
}

// wallet1 == wallet2
var Wallet1 *entropystore.Manager
var Wallet2 *entropystore.Manager
var Wallet3 *entropystore.Manager

func PreTest() {
	w := wallet.New(&wallet.Config{
		DataDir:        WalletDir,
		MaxSearchIndex: 100000,
	})
	w.Start()

	w1, err := w.GetEntropyStoreManager("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")

	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}
	err = w1.Unlock("123456")

	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}

	Wallet1 = w1

	w2, err := w.RecoverEntropyStoreFromMnemonic("extend excess vibrant crop split vehicle order veteran then fog panel appear frozen deer logic path yard tenant bag nuclear witness annual silent fold", "en", "123456", nil)

	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}
	err = w2.Unlock("123456")
	if err != nil {

		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}

	Wallet2 = w2

	w3, err := w.RecoverEntropyStoreFromMnemonic("alarm canal scheme actor left length bracket slush tuna garage prepare scout school pizza invest rose fork scorpion make enact false kidney mixed vast", "en", "123456", nil)

	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}
	err = w3.Unlock("123456")
	if err != nil {
		fmt.Errorf("wallet error, %+v", err)
		os.Exit(0)
		return
	}

	Wallet3 = w3
}

func TestWallet(t *testing.T) {
	PreTest()
	t.Log("----------------------Wallet1----------------------")
	for i := uint32(0); i < 10; i++ {
		_, key, err := Wallet1.DeriveForIndexPath(i, nil)
		if err != nil {
			t.Error(err)
			return
		}
		t.Log(key.Address())
	}
	t.Log("----------------------Wallet1----------------------")
	t.Log("----------------------Wallet2----------------------")
	for i := uint32(0); i < 10; i++ {
		_, key, err := Wallet2.DeriveForIndexPath(i, nil)
		if err != nil {
			t.Error(err)
			return
		}
		t.Log(key.Address())
	}
	t.Log("----------------------Wallet2----------------------")
	t.Log("----------------------Wallet3----------------------")
	for i := uint32(0); i < 10; i++ {
		_, key, err := Wallet3.DeriveForIndexPath(i, nil)
		if err != nil {
			t.Error(err)
			return
		}
		t.Log(key.Address())
	}
	t.Log("----------------------Wallet3----------------------")
}

func TestClient_SubmitRequestTx(t *testing.T) {
	PreTest()
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
	self, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(err)
		return
	}
	to, err := types.HexToAddress("vite_73c6b08e401608bca17272e7c59508f2e549c221ae7efccd53")
	if err != nil {
		t.Error(err)
		return
	}
	err = client.SubmitRequestTx(RequestTxParams{
		ToAddr:       to,
		SelfAddr:     self,
		Amount:       big.NewInt(10000),
		TokenId:      ledger.ViteTokenId,
		SnapshotHash: nil,
		Data:         []byte("hello pow"),
	}, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return Wallet3.SignData(addr, data, nil, nil)
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestClient_SubmitRequestTxWithPow(t *testing.T) {
	PreTest()
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
	err = client.SubmitRequestTxWithPow(RequestTxParams{
		ToAddr:       to,
		SelfAddr:     self,
		Amount:       big.NewInt(10001),
		TokenId:      ledger.ViteTokenId,
		SnapshotHash: nil,
		Data:         []byte("hello pow"),
	}, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return Wallet1.SignData(addr, data, nil, nil)
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestClient_SubmitResponseTx(t *testing.T) {
	PreTest()
	to, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
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

	for true {
		bs, err := client.QueryOnroad(OnroadQuery{
			Address: to,
			Index:   1,
			Cnt:     10,
		})
		if err != nil {
			t.Error(err)
			return
		}

		if len(bs) == 0 {
			break
		}

		for _, v := range bs {
			t.Log("receive request.", v.Hash, v.AccountAddress, v.Amount)
			err = client.SubmitResponseTx(ResponseTxParams{
				SelfAddr:     to,
				RequestHash:  v.Hash,
				SnapshotHash: nil,
			}, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
				return Wallet1.SignData(addr, data, nil, nil)
			})
			if err != nil {
				t.Error(err)
				return
			}
		}

	}
}

func TestClient_SubmitResponseTxWithPow(t *testing.T) {
	PreTest()
	to, err := types.HexToAddress("vite_73c6b08e401608bca17272e7c59508f2e549c221ae7efccd53")
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

	for true {
		bs, err := client.QueryOnroad(OnroadQuery{
			Address: to,
			Index:   1,
			Cnt:     10,
		})
		if err != nil {
			t.Error(err)
			return
		}

		if len(bs) == 0 {
			break
		}

		for _, v := range bs {
			t.Log("receive request.", v.Hash, v.AccountAddress, v.Amount)
			err = client.SubmitResponseTxWithPow(ResponseTxParams{
				SelfAddr:     to,
				RequestHash:  v.Hash,
				SnapshotHash: nil,
			}, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
				return Wallet1.SignData(addr, data, nil, nil)
			})
			if err != nil {
				t.Error(err)
				return
			}
		}

	}
}

func TestClient_QueryOnroad(t *testing.T) {
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

	addr, err := types.HexToAddress("vite_73c6b08e401608bca17272e7c59508f2e549c221ae7efccd53")
	if err != nil {
		t.Error(err)
		return
	}

	bs, err := client.QueryOnroad(OnroadQuery{
		Address: addr,
		Index:   1,
		Cnt:     10,
	})
	if e != nil {
		t.Error(e)
		return
	}
	if len(bs) > 0 {
		for _, v := range bs {
			t.Log(v)
		}
	}
}
