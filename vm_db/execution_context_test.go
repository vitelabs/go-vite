package vm_db

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"math/big"
	"testing"
)

func Test(t *testing.T) {
	addr1, _ := types.HexToAddress("vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906")
	addr2, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	data, _ := hex.DecodeString("f021ab8f0000000000000000000000000000000000000000000000000000000000000005")

	vdb := NewVmDbByAddr(nil, &addr2)
	block := &ledger.AccountBlock{
		Height:         4,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		PrevHash:       types.DataHash([]byte{0, 0}),
		Amount:         big.NewInt(1e18),
		TokenId:        ledger.ViteTokenId,
		Data:           data,
		Hash:           types.DataHash([]byte{1, 2}),
	}

	ec := ledger.ExecutionContext{
		ReferrerSendHash: types.DataHash([]byte{1, 1}),
		CallbackId: *big.NewInt(22),
	}
	vdb.SetExecutionContext(&block.Hash, &ec)

	got, _ := vdb.GetExecutionContext(&block.Hash)

	assert.Equal(t, *got, ec)
}