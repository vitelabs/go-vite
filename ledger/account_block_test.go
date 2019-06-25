package ledger

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
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
	//prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))
	prevHash, err := types.HexToHash("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		panic(err)
	}

	fromBlockHash, _ := types.HexToHash("4b4d6cf7d2f0f6ef25f8b63c1e8d58cecbf29bd3b5fb484a9b53060ecea19f34")
	//logHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is logHash")))

	addr1, err := types.HexToAddress("vite_afc922b148b3b792ecff2e79fa17255c22f15d43a77dd79f15")
	if err != nil {
		panic(err)
	}
	addr2, err := types.HexToAddress("vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a")
	if err != nil {
		panic(err)
	}

	publicKey := []byte{146, 4, 102, 210, 240, 121, 18, 183, 101, 145, 74, 10, 42, 214, 120,
		193, 131, 136, 161, 34, 13, 13, 167, 76, 142, 211, 246, 186, 111, 200, 217, 69}

	//data, err := hex.DecodeString("Y2EtNDM=")
	//if err != nil {
	//	panic(err)
	//}

	//amount, ok := big.NewInt(0).SetString("1293523228570825505871", 10)
	//if !ok {
	//	panic("err")
	//}

	//fee, ok := big.NewInt(0).SetString("10000000000000000000", 10)
	//if !ok {
	//	panic("false")
	//}

	//data, err := hex.DecodeString("EanfpRVAa7OSHYn9MSXMmicAOWVDaoHqyVKGSwVqLoA")
	//if err != nil {
	//	panic(err)
	//}
	//
	//data, err := base64.StdEncoding.DecodeString("EanfpRVAa7OSHYn9MSXMmicAOWVDaoHqyVKGSwVqLoA")
	//if err != nil {
	//	panic(err)
	//}

	//ab := &AccountBlock{}
	//json.Unmarshal([]byte(`{"blockType":4,"hash":"57410f8496598113220fd7f8780cedec5f23f13bbe0d4852e1c0e782092b3a46","prevHash":"0000000000000000000000000000000000000000000000000000000000000000","height":1,"accountAddress":"vite_afc922b148b3b792ecff2e79fa17255c22f15d43a77dd79f15","publicKey":"P8UiTllDO9PSMg8DrTt6g5MQuppfgTN7HF9A+UNUgA=","toAddress":"vite_0000000000000000000000000000000000000000a4f3a0cb58","amount":null,"tokenId":"tti_000000000000000000004cfd","fromBlockHash":"4b4d6cf7d2f0f6ef25f8b63c1e8d58cecbf29bd3b5fb484a9b53060ecea19f34","data":"EanfpRVAa7OSHYn9MSXMmicAOWVDaoHqyVKGSwVqLoA","quota":0,"quotaUsed":0,"fee":10000000000000000000,"logHash":null,"difficulty":null,"nonce":null,"sendBlockList":[],"signature":"s5puuFRSREH7eug1lAEcl9RORNNf04KvZQa0ghBgy80lVvl8K1VOp7H4vZ88nxfQecSmMP3ges91iU0RzVlDg=="}`), ab)
	//fmt.Printf("haha:%+v\n", ab)

	block := &AccountBlock{
		BlockType: BlockTypeReceive,
		PrevHash:  prevHash,
		Height:    1,

		AccountAddress: addr1,
		PublicKey:      publicKey,
		ToAddress:      addr2,

		//Amount:        amount,
		//TokenId:       ViteTokenId,
		FromBlockHash: fromBlockHash,

		Data: []byte{17, 169, 223, 165, 21, 64, 107, 179, 146, 29, 137, 253, 49, 37, 204, 154, 39, 0, 57, 101, 67, 106, 129, 234, 203, 245, 74, 25, 44, 21, 168, 186, 0},

		//Quota: 1234,
		//Fee: fee,
		//LogHash: &logHash,

		//Difficulty: big.NewInt(10),
		//Nonce:      []byte("12345678"),
	}

	str, err := json.Marshal(block)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", str)

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
