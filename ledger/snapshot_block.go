package ledger

import (
	"math/big"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
	"time"
)

var GenesisSnapshotBlockHash = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
var GenesisProducer = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}


type SnapshotBlock struct {
	// Snapshot block hash
	Hash []byte

	// Previous snapshot block hash
	PrevHash []byte

	// Height of current snapshot block
	Height *big.Int

	// Producer create the block
	Producer []byte

	// Current snapshot
	Snapshot map[string][]byte

	// Signature
	Signature []byte

	// Timestamp
	Timestamp uint64

	// Reward fee
	Amount *big.Int
}

func (sb *SnapshotBlock) DbDeserialize (buf []byte) error {
	snapshotBlockPB := &vitepb.SnapshotBlock{}
	if err := proto.Unmarshal(buf, snapshotBlockPB); err != nil {
		return err
	}
	sb.Hash = snapshotBlockPB.Hash
	sb.PrevHash = snapshotBlockPB.PrevHash
	sb.Height = &big.Int{}
	sb.Height.SetBytes(snapshotBlockPB.Height)
	sb.Producer = snapshotBlockPB.Producer
	sb.Snapshot = snapshotBlockPB.Snapshot
	sb.Signature = snapshotBlockPB.Signature
	sb.Timestamp = snapshotBlockPB.Timestamp
	sb.Amount = &big.Int{}
	sb.Amount.SetBytes(snapshotBlockPB.Amount)
	return nil
}

func (sb *SnapshotBlock) DbSerialize () ([]byte, error) {
	snapshotBlock := &vitepb.SnapshotBlock{
		Hash: sb.Hash,
		PrevHash: sb.PrevHash,
		Producer: sb.Producer,
		Snapshot: sb.Snapshot,
		Signature: sb.Signature,
		Timestamp: sb.Timestamp,
	}
	if sb.Amount != nil {
		snapshotBlock.Amount = sb.Amount.Bytes()
	}
	if sb.Height != nil {
		snapshotBlock.Height = sb.Height.Bytes()
	}
	return proto.Marshal(snapshotBlock)
}

func GetGenesisSnapshot () *SnapshotBlock {
	snapshotBLock := &SnapshotBlock{
		Hash: GenesisSnapshotBlockHash,
		PrevHash: GenesisSnapshotBlockHash,
		Height: big.NewInt(1),
		Timestamp: uint64(time.Now().Unix()),
		Producer: GenesisProducer,
	}
	return snapshotBLock
}