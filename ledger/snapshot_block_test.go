package ledger

import (
	"github.com/vitelabs/go-vite/common/types"

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

func createSnapshotBlock(scCount int) *SnapshotBlock {
	_, privateKey, _ := types.CreateAddress()
	now := time.Now()
	prevHash, _ := types.BytesToHash(crypto.Hash256([]byte("This is prevHash")))

	sb := &SnapshotBlock{
		PrevHash:        prevHash,
		Height:          123,
		PublicKey:       privateKey.PubByte(),
		Timestamp:       &now,
		SnapshotContent: createSnapshotContent(scCount),
	}
	sb.Hash = sb.ComputeHash()
	sb.Signature = ed25519.Sign(privateKey, sb.Hash.Bytes())
	return sb

}

func BmSnapshotBlockHash(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		block.ComputeHash()
	}
}

func BmSnapshotBlockSerialize(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if _, err := block.Serialize(); err != nil {
			b.Fatal(err)
		}
	}
}

func BmSnapshotBlockDeserialize(b *testing.B, scCount int) {
	b.StopTimer()
	block := createSnapshotBlock(scCount)
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
	block := createSnapshotBlock(scCount)
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
