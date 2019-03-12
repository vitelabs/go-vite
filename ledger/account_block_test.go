package ledger

import (
	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
	"testing"
)

func createBlock() *AccountBlock {
	accountAddress1, privateKey, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()

	hash, _ := types.BytesToHash(crypto.Hash256([]byte("This is hash")))
	prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))
	fromBlockHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is fromBlockHash")))
	stateHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is stateHash")))
	logHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is logHash")))

	signature := ed25519.Sign(privateKey, hash.Bytes())

	return &AccountBlock{
		BlockType: BlockTypeSendCall,
		Hash:      hash,
		PrevHash:  prevHash,
		Height:    123,

		AccountAddress: accountAddress1,
		PublicKey:      privateKey.PubByte(),
		ToAddress:      accountAddress2,

		Amount:        big.NewInt(1000),
		TokenId:       ViteTokenId,
		FromBlockHash: fromBlockHash,

		Data: []byte{'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e',
			'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e', 'a', 'b', 'c', 'd', 'e'},

		Quota:     1,
		Fee:       big.NewInt(10),
		StateHash: stateHash,
		LogHash:   &logHash,

		Difficulty: big.NewInt(10),
		Nonce:      []byte("test nonce test nonce"),
		Signature:  signature,
	}
}
func BenchmarkAccountBlock_ComputeHash(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		block.ComputeHash()
	}
}

func BenchmarkAccountBlock_Proto(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		block.proto()
	}
}

func BenchmarkAccountBlock_DeProto(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	pb := block.proto()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		newBlock := &AccountBlock{}
		if err := newBlock.deProto(pb); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAccountBlock_Serialize(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if _, err := block.Serialize(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAccountBlock_Deserialize(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	buf, err := block.Serialize()
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		newBlock := &AccountBlock{}
		if err := newBlock.Deserialize(buf); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAccountBlock_VerifySignature(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if !block.VerifySignature() {
			b.Fatal("error!")
		}
	}
}
