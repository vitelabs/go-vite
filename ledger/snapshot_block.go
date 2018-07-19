package ledger

import (
	"math/big"
	"github.com/golang/protobuf/proto"
	"time"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
)

var GenesisSnapshotBlockHash, _ = types.BytesToHash([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
var GenesisProducer = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}


type SnapshotBlockList []*SnapshotBlock
func (sblist SnapshotBlockList) NetSerialize () ([]byte, error) {
	snapshotBlockListNetPB := &vitepb.SnapshotBlockListNet{}
	snapshotBlockListNetPB.Blocks = []*vitepb.SnapshotBlock{}

	for _, snapshotBlock := range sblist {
		snapshotBlockListNetPB.Blocks = append(snapshotBlockListNetPB.Blocks, snapshotBlock.GetNetPB())
	}

	return proto.Marshal(snapshotBlockListNetPB)
}

func (sblist SnapshotBlockList) NetDeserialize (buf []byte) (error) {
	snapshotBlockListNetPB := &vitepb.SnapshotBlockListNet{}
	if err := proto.Unmarshal(buf, snapshotBlockListNetPB); err != nil {
		return err
	}

	for _, blockPB := range snapshotBlockListNetPB.Blocks {
		block := &SnapshotBlock{}
		block.SetByNetPB(blockPB)
		sblist = append(sblist, block)
	}

	return nil
}

type SnapshotBlock struct {
	// Snapshot block hash
	Hash *types.Hash

	// Previous snapshot block hash
	PrevHash *types.Hash

	// Height of current snapshot block
	Height *big.Int

	// Producer create the block
	Producer []byte

	// Current snapshot
	Snapshot map[string]*types.Hash

	// Signature
	Signature []byte

	// Timestamp
	Timestamp uint64

	// Reward fee
	Amount *big.Int
}

func (ab *SnapshotBlock) SetHash () error {
	// Hash source data:
	//var source []byte

	return nil
}

func (sb *SnapshotBlock) GetDbPB () (*vitepb.SnapshotBlock) {
	snapshotBlockPB := &vitepb.SnapshotBlock{
		Producer: sb.Producer,
		Signature: sb.Signature,
		Timestamp: sb.Timestamp,
	}
	if sb.Hash != nil {
		snapshotBlockPB.Hash = sb.Hash.Bytes()
	}

	if sb.PrevHash != nil {
		snapshotBlockPB.PrevHash = sb.PrevHash.Bytes()
	}


	if sb.Snapshot != nil {
		snapshotBlockPB.Snapshot = sb.GetSnapshotPB()
	}
	if sb.Amount != nil {
		snapshotBlockPB.Amount = sb.Amount.Bytes()
	}
	if sb.Height != nil {
		snapshotBlockPB.Height = sb.Height.Bytes()
	}

	return snapshotBlockPB
}

func (sb *SnapshotBlock) SetByDbPB (snapshotBlockPB *vitepb.SnapshotBlock) (error) {
	if snapshotBlockPB.Hash != nil {
		hash, err := types.BytesToHash(snapshotBlockPB.Hash)
		if err != nil {
			return err
		}
		sb.Hash = &hash
	}
	if snapshotBlockPB.PrevHash != nil {
		prevHash, err := types.BytesToHash(snapshotBlockPB.PrevHash)
		if err != nil {
			return err
		}
		sb.PrevHash = &prevHash
	}

	sb.Height = &big.Int{}
	sb.Height.SetBytes(snapshotBlockPB.Height)
	sb.Producer = snapshotBlockPB.Producer
	if snapshotBlockPB.Snapshot != nil {
		err := sb.SetSnapshotByPB(snapshotBlockPB.Snapshot)
		if err != nil {
			return err
		}
	}
	sb.Signature = snapshotBlockPB.Signature
	sb.Timestamp = snapshotBlockPB.Timestamp
	sb.Amount = &big.Int{}
	sb.Amount.SetBytes(snapshotBlockPB.Amount)
	return nil
}


func (sb *SnapshotBlock) GetNetPB () (*vitepb.SnapshotBlock) {
	return sb.GetDbPB()
}

func (sb *SnapshotBlock) SetByNetPB (pb *vitepb.SnapshotBlock) (error) {
	return sb.SetByDbPB(pb)
}

func (sb *SnapshotBlock) GetSnapshotPB () (map[string][]byte) {
	snapshotPB := make(map[string][]byte)
	for key, hash := range sb.Snapshot {
		snapshotPB[key] = hash.Bytes()
	}

	return snapshotPB
}

func (sb *SnapshotBlock) SetSnapshotByPB (snapshotPB map[string][]byte) (error) {
	sb.Snapshot = make(map[string]*types.Hash)
	for key, hashBytes := range snapshotPB {
		hash, err := types.BytesToHash(hashBytes)
		if err != nil {
			return err
		}
		sb.Snapshot[key] = &hash
	}

	return nil
}


func (sb *SnapshotBlock) DbDeserialize (buf []byte) error {
	snapshotBlockPB := &vitepb.SnapshotBlock{}
	if err := proto.Unmarshal(buf, snapshotBlockPB); err != nil {
		return err
	}

	sb.SetByDbPB(snapshotBlockPB)

	return nil
}

func (sb *SnapshotBlock) DbSerialize () ([]byte, error) {
	return proto.Marshal(sb.GetDbPB())
}

func GetGenesisSnapshot () *SnapshotBlock {
	snapshotBLock := &SnapshotBlock{
		Hash: &GenesisSnapshotBlockHash,
		PrevHash: &GenesisSnapshotBlockHash,
		Height: big.NewInt(1),
		Timestamp: uint64(time.Now().Unix()),
		Producer: GenesisProducer,
	}
	return snapshotBLock
}