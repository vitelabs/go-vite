package ledger

import (
	"encoding/base64"
	"encoding/json"

	"fmt"
	"github.com/vitelabs/go-vite/common/fork"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"

	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/rand"
	"testing"
	"time"
)

func createSnapshotContent(count int) SnapshotContent {
	sc := make(SnapshotContent, count)
	for i := 0; i < count; i++ {
		addr, _, _ := types.CreateAddress()
		height := rand.Uint64()

		hash, _ := types.BytesToHash(crypto.Hash256(addr.Bytes()))
		sc[addr] = &HashHeight{
			Hash:   hash,
			Height: height,
		}
	}
	return sc
}

func createSnapshotBlock(scCount int, sbheight uint64) *SnapshotBlock {
	_, privateKey, _ := types.CreateAddress()
	now := time.Now()
	prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))

	sb := &SnapshotBlock{
		PrevHash:        prevHash,
		Height:          sbheight,
		PublicKey:       privateKey.PubByte(),
		Timestamp:       &now,
		SnapshotContent: createSnapshotContent(scCount),
	}
	sb.Hash = sb.ComputeHash()
	sb.Signature = ed25519.Sign(privateKey, sb.Hash.Bytes())
	return sb

}

func TestSignature(t *testing.T) {
	hash, err := types.HexToHash("5835374c6b5b612e5016ae04689d57f5c49e3a520412121607a28566d1c62825")
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString("4DxxPWO/hymiRBM36rFsv82AXo59QKHw4dMrlQSJTCU=")
	if err != nil {
		t.Fatal(err)
	}

	signatureBytes, err := base64.StdEncoding.DecodeString("x/pYNm+qMGB0npHW1IIbnCKbczJLFQcjC7zyulEGA8IMvq8sxhWBbRYFd10SypubKevVhmdnrKV86UBIGovADQ==")
	if err != nil {
		t.Fatal(err)
	}

	sb := &SnapshotBlock{
		Hash:      hash,
		PublicKey: pubKeyBytes,
		Signature: signatureBytes,
	}
	fmt.Printf("%+v\n", sb)
	bytes, err := json.Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", bytes)

	fmt.Println(sb.VerifySignature())

	privKey, err := ed25519.HexToPrivateKey("949b71dd87bb239f1916365186c075b40307d15259aa331811c3ecf7557e61bee03c713d63bf8729a2441337eab16cbfcd805e8e7d40a1f0e1d32b9504894c25")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(ed25519.Sign(privKey, hash.Bytes())))
	fmt.Println(base64.StdEncoding.EncodeToString(privKey.PubByte()))

	fmt.Println(crypto.VerifySig(privKey.PubByte(), sb.Hash.Bytes(), ed25519.Sign(privKey, hash.Bytes())))

}

func BmSnapshotBlockHash(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount, 123)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		block.ComputeHash()
	}
}

func BmSnapshotBlockSerialize(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount, 123)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if _, err := block.Serialize(); err != nil {
			b.Fatal(err)
		}
	}
}

func BmSnapshotBlockDeserialize(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount, 123)
	buf, err := block.Serialize()
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		newBlock := &SnapshotBlock{}
		if err := newBlock.Deserialize(buf); err != nil {
			b.Fatal(err)
		}
	}
}

func BmSnapshotBlockVerifySignature(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount, 123)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if ok := block.VerifySignature(); !ok {
			b.Fatal("error")
		}
	}
}

func BenchmarkSnapshotBlock_ComputeHash(b *testing.B) {
	b.Run("snapshot 10 accounts", func(b *testing.B) {
		BmSnapshotBlockHash(b, 10)
	})
	b.Run("snapshot 100 accounts", func(b *testing.B) {
		BmSnapshotBlockHash(b, 100)
	})
	b.Run("snapshot 1000 accounts", func(b *testing.B) {
		BmSnapshotBlockHash(b, 1000)
	})
	b.Run("snapshot 10000 accounts", func(b *testing.B) {
		BmSnapshotBlockHash(b, 10000)
	})
	b.Run("snapshot 100000 accounts", func(b *testing.B) {
		BmSnapshotBlockHash(b, 100000)
	})
}

func BenchmarkSnapshotBlock_Serialize(b *testing.B) {
	b.Run("snapshot 10 accounts", func(b *testing.B) {
		BmSnapshotBlockSerialize(b, 10)
	})
	b.Run("snapshot 100 accounts", func(b *testing.B) {
		BmSnapshotBlockSerialize(b, 100)
	})
	b.Run("snapshot 1000 accounts", func(b *testing.B) {
		BmSnapshotBlockSerialize(b, 1000)
	})
	b.Run("snapshot 10000 accounts", func(b *testing.B) {
		BmSnapshotBlockSerialize(b, 10000)
	})
	b.Run("snapshot 100000 accounts", func(b *testing.B) {
		BmSnapshotBlockSerialize(b, 100000)
	})
}

func BenchmarkSnapshotBlock_Deserialize(b *testing.B) {
	b.Run("snapshot 10 accounts", func(b *testing.B) {
		BmSnapshotBlockDeserialize(b, 10)
	})
	b.Run("snapshot 100 accounts", func(b *testing.B) {
		BmSnapshotBlockDeserialize(b, 100)
	})
	b.Run("snapshot 1000 accounts", func(b *testing.B) {
		BmSnapshotBlockDeserialize(b, 1000)
	})
	b.Run("snapshot 10000 accounts", func(b *testing.B) {
		BmSnapshotBlockDeserialize(b, 10000)
	})
	b.Run("snapshot 100000 accounts", func(b *testing.B) {
		BmSnapshotBlockDeserialize(b, 100000)
	})
}

func BenchmarkSnapshotBlock_VerifySignature(b *testing.B) {
	b.Run("snapshot 10 accounts", func(b *testing.B) {
		BmSnapshotBlockVerifySignature(b, 10)
	})
	b.Run("snapshot 100 accounts", func(b *testing.B) {
		BmSnapshotBlockVerifySignature(b, 100)
	})
	b.Run("snapshot 1000 accounts", func(b *testing.B) {
		BmSnapshotBlockVerifySignature(b, 1000)
	})
	b.Run("snapshot 10000 accounts", func(b *testing.B) {
		BmSnapshotBlockVerifySignature(b, 10000)
	})
	b.Run("snapshot 100000 accounts", func(b *testing.B) {
		BmSnapshotBlockVerifySignature(b, 100000)
	})
}

func TestForkComputeHash(t *testing.T) {

	snapshotBlock := createSnapshotBlock(1, 10000000000000)
	hashold := snapshotBlock.Hash
	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{
			Height:  90,
			Version: 1,
		},
	})

	hashnew := snapshotBlock.ComputeHash()

	if hashold == hashnew {
		t.Fatal(fmt.Sprintf("is not right, old: %+v,  new:  %+v", hashold, hashnew))
	}

	fmt.Println("old and new hash:", hashold, hashnew)

}
