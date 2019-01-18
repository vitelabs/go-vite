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
	addr, _ := types.HexToAddress("vite_c2406893861fa23cc5deafb2461f50163b728ccc2d2931b830")
	//nonce, _ := base64.StdEncoding.DecodeString("PRdIJ3eSXDQ=")
	preHash, _ := types.HexToHash("1d3eaf71c77d6e219f42743347bbc3e24bed3b458b2b39a3ac606fc0778f741d")
	snapshotBlockHash, _ := types.HexToHash("8584c529b76ec0fca65e7934d93d7e6e5fe4d19be006b17cd40c3150289b7b5a")
	data, _ := base64.StdEncoding.DecodeString("8J+Qu/CfkKjwn5Cx")
	a, _ := new(big.Int).SetString("11000000000000000000", 10)
	toaddr, _ := types.HexToAddress("vite_f505b7e70389bb14db9b6cb84ea41bd7b822a71864b659445f")
	tti, _ := types.HexToTokenTypeId("tti_c55ec37a916b7f447575ae59")
	ts := time.Unix(1539604021, 0)
	block := &AccountBlock{
		BlockType: 2,

		Amount:         a,
		Height:         6,
		PrevHash:       preHash,
		AccountAddress: addr,
		Fee:            big.NewInt(0),
		ToAddress:      toaddr,
		TokenId:        tti,
		Timestamp:      &ts,
		Data:           data,
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
