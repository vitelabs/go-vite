package client

import (
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/wallet"
)

func TestClient(t *testing.T) {
	w := wallet.New(&wallet.Config{
		DataDir:        "~/Library/GVite/devdata/wallet",
		MaxSearchIndex: 100000,
	})

	w.Start()

	manager, err := w.GetEntropyStoreManager("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")

	if err != nil {
		t.Error(err)
		return
	}
	manager.Unlock("123456")
	path, key, err := manager.DeriveForIndexPath(0, nil)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(path)
	t.Log(key.Address())

	em, err := w.RecoverEntropyStoreFromMnemonic("extend excess vibrant crop split vehicle order veteran then fog panel appear frozen deer logic path yard tenant bag nuclear witness annual silent fold", "en", "123456", nil)

	if err != nil {
		t.Error(err)
		return
	}
	err = em.Unlock("123456")
	if err != nil {
		t.Error(err)
		return
	}
	path, key, err = em.DeriveForIndexPath(0, nil)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(path)
	t.Log(key.Address())
}

func TestClient_SubmitRequestTx(t *testing.T) {
	w := wallet.New(&wallet.Config{
		DataDir:        "~/Library/GVite/devdata/wallet",
		MaxSearchIndex: 100000,
	})

	w.Start()
	em, err := w.RecoverEntropyStoreFromMnemonic("alarm canal scheme actor left length bracket slush tuna garage prepare scout school pizza invest rose fork scorpion make enact false kidney mixed vast", "en", "123456", nil)

	if err != nil {
		t.Error(err)
		return
	}
	err = em.Unlock("123456")
	if err != nil {
		t.Error(err)
		return
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
	self, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(err)
		return
	}
	to, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
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
		Data:         nil,
	}, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return em.SignData(addr, data, nil, nil)
	})
	if err != nil {
		t.Error(err)
		return
	}
}
func TestClient_SubmitResponseTx(t *testing.T) {
	w := wallet.New(&wallet.Config{
		DataDir:        "~/Library/GVite/devdata/wallet",
		MaxSearchIndex: 100000,
	})

	w.Start()
	em, err := w.RecoverEntropyStoreFromMnemonic("alarm canal scheme actor left length bracket slush tuna garage prepare scout school pizza invest rose fork scorpion make enact false kidney mixed vast", "en", "123456", nil)

	if err != nil {
		t.Error(err)
	}

	err = em.Unlock("123456")
	if err != nil {
		t.Error(err)
		return
	}

	to, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
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
				return em.SignData(addr, data, nil, nil)
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

	addr, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
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
