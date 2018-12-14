package client

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
)

var RawUrl = "http://127.0.0.1:48132"

//var RawUrl = "http://45.40.197.46:48132"

func TestSendRaw(t *testing.T) {
	//client, err := NewRpcClient("http://45.40.197.46:48132")
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	accountAddress, _ := types.HexToAddress("vite_00000000000000000000000000000000000000056ad6d26692")
	toAddress, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	tokenId, _ := types.HexToTokenTypeId("tti_3cd880a76b7524fc2694d607")
	snapshotHash, _ := types.HexToHash("68d458d52a13d5594c069a365345d2067ccbceb63680ec384697dda88de2ada8")
	publicKey, _ := hex.DecodeString("4sYVHCR0fnpUZy3Acj8Wy0JOU81vH/khAW1KLYb19Hk=")

	amount := big.NewInt(1000000000).String()
	block := RawBlock{
		BlockType: 3,
		//PrevHash:       prevHash,
		AccountAddress: accountAddress,
		PublicKey:      publicKey,
		ToAddress:      toAddress,
		TokenId:        tokenId,
		SnapshotHash:   snapshotHash,
		Height:         "6",
		Amount:         &amount,
		Timestamp:      time.Now().Unix(),
	}
	err = client.SubmitRaw(block)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestFittest(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	hashes, e := client.GetFittestSnapshot()
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(hashes)
}

func TestGetSnapshotByHeight(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	hash, err := types.HexToHash("b3725777f3b8a3c6a1d126a934e0757d9b9e55df791639b2e241c78c75b8137f")
	if err != nil {
		t.Fatal(err)
	}

	h, e := client.GetSnapshotByHash(hash)
	if e != nil {
		t.Error(e)
		return
	}

	t.Log(h)

	heightInt, e := strconv.ParseUint(h.Height, 10, 64)
	if e != nil {
		t.Fatal(e)
	}
	h2, e := client.GetSnapshotByHeight(heightInt)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(h2)
}

func TestAccBlock(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}
	hash, err := types.HexToHash("bfff83c40823c60ff8b28430f988334e60f49a9adacfc4b94b2fce224aa97d14")
	if err != nil {
		t.Error(err)
		return
	}
	block, e := client.GetAccBlock(hash)
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(block)
	t.Log(block.TokenId)
}

func TestQueryOnroad(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	addr, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
	if err != nil {
		t.Error(addr)
		return
	}
	bs, e := client.GetOnroad(OnroadQuery{
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

func TestQueryBalance(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	addr, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
	if err != nil {
		t.Error(addr)
		return
	}
	bs, e := client.Balance(BalanceQuery{
		Addr:    addr,
		TokenId: ledger.ViteTokenId,
	})
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(bs)
}

func TestQueryBalanceAll(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	addr, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
	if err != nil {
		t.Error(addr)
		return
	}
	bs, e := client.BalanceAll(BalanceAllQuery{
		Addr: addr,
	})
	if e != nil {
		t.Error(e)
		return
	}
	for _, v := range bs {
		t.Log(v)
	}

}
