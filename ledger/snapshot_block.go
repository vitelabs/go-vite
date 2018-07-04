package ledger

import (
	"math/big"
)

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

	// Reward fee
	Amount *big.Int
}

func (sb *SnapshotBlock) DbDeserialize (buf []byte) error {
	return nil
}

//func (sb * SnapshotBlock) Serialize () ([]byte, error) {
//	snapshotBlockPB := & vitepb.SnapshotBlock{
//		PrevHash: sb.PrevHash,
//
//		BlockNum: sb.BlockNum.Bytes(),
//
//		Producer: sb.Producer,
//
//		Snapshot: sb.Snapshot,
//
//		Signature: sb.Signature,
//	}
//
//	serializedBytes, err := proto.Marshal(snapshotBlockPB)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return serializedBytes, nil
//}
//
//func (sb * SnapshotBlock) Deserialize (buf []byte) error {
//	snapshotBlockPB := &vitepb.SnapshotBlock{}
//	if err := proto.Unmarshal(buf, snapshotBlockPB); err != nil {
//		return err
//	}
//
//	sb.PrevHash = snapshotBlockPB.PrevHash
//
//	sb.BlockNum = &big.Int{}
//	sb.BlockNum.SetBytes(snapshotBlockPB.BlockNum)
//
//	sb.Producer = snapshotBlockPB.Producer
//
//	sb.Snapshot = snapshotBlockPB.Snapshot
//
//	sb.Signature = snapshotBlockPB.Signature
//
//	return nil
//}