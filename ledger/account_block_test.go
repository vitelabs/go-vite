package ledger

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"

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
