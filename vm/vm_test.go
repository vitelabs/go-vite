package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
)

func TestRun(t *testing.T) {
	vm := &VM{Db: NewNoDatabase(), createBlock: CreateNoAccountBlock, instructionSet: simpleInstructionSet}
	vm.Debug = true
	// return 1+2
	inputdata, _ := hex.DecodeString("6001600201602080919052602090F3")
	receiveCallBlock := CreateNoAccountBlock(types.Address{}, types.Address{}, BlockTypeReceive, 1)
	c := newContract(receiveCallBlock.AccountAddress(), receiveCallBlock.ToAddress(), receiveCallBlock, 1000000, 0)
	c.setCallCode(types.Address{}, inputdata)
	ret, _ := c.run(vm)
	expectedRet, _ := hex.DecodeString("03")
	expectedRet = leftPadBytes(expectedRet, 32)
	if bytes.Compare(ret, expectedRet) != 0 || c.quotaLeft != 999964 || c.quotaRefund != 0 {
		t.Fatalf("expected [%v], get [%v]", expectedRet, ret)
	}
}

func TestVM_CreateSend(t *testing.T) {
	inputdata, _ := hex.DecodeString("608060405260008055348015601357600080fd5b5060358060216000396000f3006080604052600080fd00a165627a7a723058207c31c74808fe0f95820eb3c48eac8e3e10ef27058dc6ca159b547fccde9290790029")
	sendCreateBlock := CreateNoAccountBlock(types.Address{}, types.Address{}, BlockTypeSendCreate, 1)
	sendCreateBlock.SetTokenId(viteTokenTypeId)
	sendCreateBlock.SetAmount(big.NewInt(0))
	sendCreateBlock.SetSnapshotHash(types.Hash{})
	sendCreateBlock.SetPrevHash(types.Hash{})
	sendCreateBlock.SetHeight(big.NewInt(1))
	sendCreateBlock.SetData(inputdata)
	sendCreateBlock.SetCreateFee(big.NewInt(0))
	// vm.Debug = true
	vm := NewVM(NewNoDatabase(), CreateNoAccountBlock)
	blockList, _, err := vm.Run(sendCreateBlock)
	if len(blockList) != 1 ||
		//blockList[0].Quota() != 58336 ||
		blockList[0].ToAddress() == emptyAddress ||
		//blockList[0].Balance() == nil ||
		blockList[0].Amount().Cmp(big.NewInt(0)) != 0 ||
		//blockList[0].StateHash() == emptyHash ||
		blockList[0].TokenId() != viteTokenTypeId {
		t.Fatalf("send create fail [%v] %v", blockList, err)
	}
}

func TestGetTokenInfo(t *testing.T) {
	tests := []struct {
		tokenTypeId types.TokenTypeId
		owner       types.Address
		totalSupply *big.Int
		data        string
		result      error
	}{
		{types.CreateTokenTypeId([]byte{1}), types.CreateContractAddress([]byte{1}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", nil},
		{types.CreateTokenTypeId([]byte{2}), types.CreateContractAddress([]byte{2}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{3}), types.CreateContractAddress([]byte{3}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{4}), types.CreateContractAddress([]byte{4}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000013000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{5}), types.CreateContractAddress([]byte{5}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001556697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{6}), types.CreateContractAddress([]byte{6}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001456697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{7}), types.CreateContractAddress([]byte{7}), big.NewInt(10), "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000093c697465546f6b656e0000000000000000000000000000000000000000000000", ErrInvalidTokenData},
		{types.CreateTokenTypeId([]byte{1}), types.CreateContractAddress([]byte{8}), big.NewInt(10), "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000956697465546f6b656e0000000000000000000000000000000000000000000000", ErrTokenIdCollision},
	}
	vm := &VM{Db: NewNoDatabase(), createBlock: CreateNoAccountBlock, instructionSet: simpleInstructionSet}
	for i, test := range tests {
		inputdata, _ := hex.DecodeString(test.data)
		err := vm.checkAndCreateToken(test.tokenTypeId, test.owner, test.totalSupply, inputdata)
		if err != test.result {
			t.Fatalf("%v th check token data fail %v %v", i, test, err)
		}
	}
}

/*func TestVM_CreateReceive(t *testing.T) {
	inputdata, _ := hex.DecodeString("608060405260008055348015601357600080fd5b5060358060216000396000f3006080604052600080fd00a165627a7a723058207c31c74808fe0f95820eb3c48eac8e3e10ef27058dc6ca159b547fccde9290790029")
	receiveCreateBlock := CreateNoAccountBlock(types.Address{}, types.Address{}, BlockTypeReceive, 1)
	receiveCreateBlock.SetTokenId(viteTokenTypeId)
	receiveCreateBlock.SetAmount(big.NewInt(0))
	receiveCreateBlock.SetSnapshotHash(types.Hash{})
	receiveCreateBlock.SetPrevHash(types.Hash{})
	receiveCreateBlock.SetHeight(big.NewInt(1))
	receiveCreateBlock.SetData(inputdata)
	vm := NewVM(&NoDatabase{}, CreateNoAccountBlock)
	vm.Debug = true
	blockList, err := vm.Run(receiveCreateBlock)
	if len(blockList) != 1 ||
		//blockList[0].Quota() != 89008 ||
		//blockList[0].Balance() == nil ||
		//blockList[0].StateHash() == emptyHash ||
		blockList[0].Amount().Cmp(big.NewInt(0)) != 0 ||
		blockList[0].TokenId() != viteTokenTypeId {
		t.Fatalf("send create fail [%v] %v", blockList, err)
	}
}*/
