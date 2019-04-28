package ledger

import (
	"github.com/vitelabs/go-vite/common/types"

	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
	"testing"
)

type SSS struct {
	A uint64
}

func TestAccountBlock_Copymap(t *testing.T) {
	Aslice := make([]SSS, 3)
	b := SSS{
		A: 2,
	}

	fmt.Println(b)
	Aslice[2] = b

	fmt.Println(Aslice[2])
	d := Aslice[2]
	d.A = 100

	b.A = 1000

	fmt.Println(d)
	fmt.Println(b)

	fmt.Println(Aslice[2])
	Aslice[2].A = 1000
	fmt.Println(Aslice[2])

}

func createBlock() *AccountBlock {
	accountAddress1, privateKey, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()

	hash, _ := types.BytesToHash(crypto.Hash256([]byte("This is hash")))
	prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))
	fromBlockHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is fromBlockHash")))
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

		Data: []byte{'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e',
			'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e', 'c', 'b', 'c', 'd', 'e'},

		Quota:   1,
		Fee:     big.NewInt(10),
		LogHash: &logHash,

		Difficulty: big.NewInt(10),
		Nonce:      []byte("test nonce test nonce"),
		Signature:  signature,
	}
}

func TestAccountBlock_ComputeHash(t *testing.T) {
	prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))
	fromBlockHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is fromBlockHash")))
	logHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is logHash")))

	addr1, _ := types.HexToAddress("vite_40ecd068e6919694d989866e3362c557984fd2637671219def")
	addr2, _ := types.HexToAddress("vite_aa01c78289d51862026d93c98115e4b540b800a877aa98a76b")

	publicKey := []byte{146, 4, 102, 210, 240, 121, 18, 183, 101, 145, 74, 10, 42, 214, 120,
		193, 131, 136, 161, 34, 13, 13, 167, 76, 142, 211, 246, 186, 111, 200, 217, 69}

	block := &AccountBlock{
		BlockType: BlockTypeSendCall,
		PrevHash:  prevHash,
		Height:    123,

		AccountAddress: addr1,
		PublicKey:      publicKey,
		ToAddress:      addr2,

		Amount:        big.NewInt(1000),
		TokenId:       ViteTokenId,
		FromBlockHash: fromBlockHash,

		Data: []byte("test data test data"),

		Quota:   1234,
		Fee:     big.NewInt(10),
		LogHash: &logHash,

		Difficulty: big.NewInt(10),
		Nonce:      []byte("12345678"),
	}

	if block.ComputeHash().String() != "6d54436d78a3bae0b4aacbeb91a0af3c666c6ed3339fbcc6610e12844736d091" {
		t.Fatal(block.ComputeHash().String())
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
		block.Proto()
	}
}

func BenchmarkAccountBlock_DeProto(b *testing.B) {
	b.StopTimer()
	block := createBlock()
	pb := block.Proto()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		newBlock := &AccountBlock{}
		if err := newBlock.DeProto(pb); err != nil {
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
