package ledger

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"

	"encoding/base64"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
	"testing"
	"time"
)

type Bclass struct {
	HAHA uint64
}

type Aclass struct {
	b  Bclass
	Ts []uint64
}

func TestComputeHash(t *testing.T) {
	addr, _ := types.HexToAddress("vite_847e1672c9a775ca0f3c3a2d3bf389ca466e5501cbecdb7107")
	nonce, _ := base64.StdEncoding.DecodeString("PRdIJ3eSXDQ=")
	fromBlockHash, _ := types.HexToHash("48290760a0249c28e92bfbcac31e1c0b61e74f666bddc1a2574b96a7bb533852")
	snapshotBlockHash, _ := types.HexToHash("3e3393b720679ff09dbc57f6e23570dbca3dc947cf28cdcbad3abc1cb6da2bee")
	ts := time.Unix(1539604021, 0)
	block := &AccountBlock{
		BlockType: 4,

		Height:         1,
		PrevHash:       types.Hash{},
		AccountAddress: addr,
		Fee:            big.NewInt(0),
		Nonce:          nonce,
		Timestamp:      &ts,
		FromBlockHash:  fromBlockHash,
		SnapshotHash:   snapshotBlockHash,
	}
	fmt.Println(block.ComputeHash())
}

func TestHash(t *testing.T) {
	source := []byte("050697d3810c30816b005a03511c734c1159f5090000000000000000000000000000000000000000000000000000000000000000")

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	fmt.Println(hash.String())
}

func TestAccountBlock_Copy(t *testing.T) {
	a := Aclass{
		b: Bclass{
			HAHA: 12,
		},
		Ts: []uint64{1, 2, 3},
	}
	fmt.Println(a.Ts)

	d := a
	fmt.Println(d.Ts)

	d.Ts[0] = 10
	fmt.Println(d.Ts)

	fmt.Println(a.Ts)

}

type RpcAccountBlock struct {
	*AccountBlock

	Height         string
	Data           string
	ConfirmedTimes uint64
}

func createAccountBlock(ledgerBlock *AccountBlock, confirmedTimes uint64) *RpcAccountBlock {
	return &RpcAccountBlock{
		AccountBlock:   ledgerBlock,
		ConfirmedTimes: confirmedTimes,
	}
}

func TestCreateAccountBlock(t *testing.T) {
	accountAddress1, privateKey, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()

	now := time.Now()

	block := &AccountBlock{
		PrevHash:       types.Hash{},
		BlockType:      BlockTypeSendCall,
		AccountAddress: accountAddress1,
		ToAddress:      accountAddress2,
		Amount:         big.NewInt(1000),
		TokenId:        ViteTokenId,
		Height:         123,
		Quota:          1,
		Fee:            big.NewInt(0),
		PublicKey:      privateKey.PubByte(),
		SnapshotHash:   types.Hash{},
		Timestamp:      &now,
		Data:           []byte{'a', 'b', 'c', 'd', 'e'},
		StateHash:      types.Hash{},
		LogHash:        &types.Hash{},
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	rpcBlock := createAccountBlock(block, 12)
	rpcBlock.Height = "1231231"
	rpcBlock.Data = "12312312312312asdijfasd"
	result, _ := json.Marshal(rpcBlock)
	fmt.Println(string(result))
}
