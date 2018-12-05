package client

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

//var RawUrl = "http://127.0.0.1:48132"
var RawUrl = "http://45.40.197.46:48132"

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

	hashes, e := client.getFittestSnapshot()
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(hashes)
}

func TestLatest(t *testing.T) {
	client, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	addr, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	if err != nil {
		t.Error(addr)
		return
	}
	hashes, e := client.GetLatest(addr)
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(hashes)
}
