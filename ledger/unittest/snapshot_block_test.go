package ledger_unittest

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/node/unittest"
	"testing"
	"time"
)

//func TestGetGenesesSnapshotBlock(t *testing.T) {
//
//}
func TestA(t *testing.T) {
	fmt.Println(ledger.ViteTokenId.String())
}

func BenchmarkGetGenesisSnapshotBlock(b *testing.B) {

	aBytes := []byte{123, 23, 224}
	for i := 0; i < 100000000; i++ {
		var aTime = time.Unix(12123123123133123, 0)
		noThing(bytes.Equal(aBytes, []byte(string(aTime.Unix()))))
	}

}

func noThing(interface{}) {

}

func TestSnapshotBlock_ComputeHash(t *testing.T) {
	// init fork config
	fork.SetForkPoints(node_unittest.MakeTestNetForkPointsConfig())

	hash, _ := types.HexToHash("e7f7728b12df832903130624a0fb72eaeee2a4d9c0e7d28d5d206459eb7f1a46")

	prevHash, _ := types.HexToHash("bdaabc5f4eda27528c0f080354c66522ce849076328337832fc3b6e575b0e855")
	height := uint64(1212922)
	publicKey, _ := ed25519.HexToPublicKey("6WNQQo98Ju8bJWgWzOYtr5ER8Oy85j")
	stateHash, _ := types.HexToHash("cca5fc60c1d1e103127952fffef0994a6e7b3d89310a1423de7ae223ec639bab")
	timestamp := time.Unix(1550578308, 0)

	addr1, _ := types.HexToAddress("vite_00000000000000000000000000000000000000056ad6d26692")
	addr1Hash, _ := types.HexToHash("41de9e174e848bea771ffe6386afc47dd2384ffb952e27bc27122d257a7bf7c5")

	addr2, _ := types.HexToAddress("vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23")
	addr2Hash, _ := types.HexToHash("acd6512bcc2ba80bc6205763d1214942b75263d83a7631f5c2a22378b4878817")

	sb := &ledger.SnapshotBlock{
		PrevHash:  prevHash,
		Height:    height,
		PublicKey: publicKey,
		StateHash: stateHash,
		Timestamp: &timestamp,
		SnapshotContent: ledger.SnapshotContent{
			addr1: {
				Height: 15,
				Hash:   addr1Hash,
			},
			addr2: {
				Height: 534,
				Hash:   addr2Hash,
			},
		},
	}
	for i := 0; i < 10000; i++ {
		computedHash := sb.ComputeHash()
		if computedHash != hash {
			t.Fatal(fmt.Sprintf("%d. computedHash != hash, computedHash is %s, hash is %s", i, computedHash, hash))
		} else {
			fmt.Printf("%d. computedHash is %s, hash is %s\n", i, computedHash, hash)
		}
	}
}
