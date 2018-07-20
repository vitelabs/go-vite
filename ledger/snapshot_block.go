package ledger

import (
	"math/big"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)


type SnapshotBlockList []*SnapshotBlock
func (sblist SnapshotBlockList) NetSerialize () ([]byte, error) {
	snapshotBlockListNetPB := &vitepb.SnapshotBlockListNet{}
	snapshotBlockListNetPB.Blocks = []*vitepb.SnapshotBlockNet{}

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

type SnapshotItem struct {
	AccountBlockHash *types.Hash
	AccountBlockHeight *big.Int
}

type SnapshotBlock struct {
	// Snapshot block hash
	Hash *types.Hash

	// Previous snapshot block hash
	PrevHash *types.Hash

	// Height of current snapshot block
	Height *big.Int

	// Producer create the block
	Producer *types.Address

	// Current snapshot
	Snapshot map[string]*SnapshotItem

	// Signature
	Signature []byte

	// Timestamp
	Timestamp uint64

	// Reward fee
	Amount *big.Int

	// Public Key
	PublicKey ed25519.PublicKey
}

func (sb *SnapshotBlock) getSnapshotBytes () []byte {
	var source []byte
	for addr, snapshotItem := range sb.Snapshot {
		address, _ := types.HexToAddress(addr)
		source = append(source, address.Bytes()...)
		source = append(source, snapshotItem.AccountBlockHash.Bytes()...)
		source = append(source, snapshotItem.AccountBlockHeight.Bytes()...)
	}

	return source
}

func (sb *SnapshotBlock) SetHash () error {
	// Hash source data:
	// PrevHash|Height|Producer|Snapshot|Timestamp|Amount
	var source []byte
	source = append(source, sb.PrevHash.Bytes()...)
	source = append(source, []byte(sb.Height.String())...)
	source = append(source, []byte(sb.Producer.String())...)
	source = append(source, sb.getSnapshotBytes()...)


	source = append(source, []byte(string(sb.Timestamp))...)
	source = append(source, []byte(sb.Amount.String())...)


	hash, err := types.BytesToHash(crypto.Hash(len(source), source))
	if err != nil {
		return err
	}

	sb.Hash = &hash
	return nil
}


func (sb *SnapshotBlock) GetNetPB () (*vitepb.SnapshotBlockNet) {
	snapshotBlockPB := &vitepb.SnapshotBlockNet{
		Signature: sb.Signature,
		Timestamp: sb.Timestamp,
		PublicKey: []byte(sb.PublicKey),
	}

	if sb.Producer != nil {
		snapshotBlockPB.Producer = sb.Producer.Bytes()
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

func (sb *SnapshotBlock) SetByNetPB (snapshotBlockPB *vitepb.SnapshotBlockNet) (error) {
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
	if snapshotBlockPB.Producer != nil {
		producer , _ := types.BytesToAddress(snapshotBlockPB.Producer)
		sb.Producer = &producer
	}
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
	sb.PublicKey = snapshotBlockPB.PublicKey
	return nil
}



func (sb *SnapshotBlock) GetDbPB () (*vitepb.SnapshotBlock) {
	snapshotBlockPB := &vitepb.SnapshotBlock{
		Signature: sb.Signature,
		Timestamp: sb.Timestamp,
	}

	if sb.Producer != nil {
		snapshotBlockPB.Producer = sb.Producer.Bytes()
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
	if snapshotBlockPB.Producer != nil {
		producer , _ := types.BytesToAddress(snapshotBlockPB.Producer)
		sb.Producer = &producer
	}
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


func (sb *SnapshotBlock) GetSnapshotPB () (map[string]*vitepb.SnapshotItem) {
	snapshotPB := make(map[string]*vitepb.SnapshotItem)
	for key, snapshotItem := range sb.Snapshot {
		snapshotPB[key] = &vitepb.SnapshotItem {
			AccountBlockHash: snapshotItem.AccountBlockHash.Bytes(),
			AccountBlockHeight: snapshotItem.AccountBlockHeight.Bytes(),
		}
	}

	return snapshotPB
}

func (sb *SnapshotBlock) SetSnapshotByPB (snapshotPB map[string]*vitepb.SnapshotItem) (error) {
	sb.Snapshot = make(map[string]*SnapshotItem)

	for key, snapshotItem := range snapshotPB {
		hash, err := types.BytesToHash(snapshotItem.AccountBlockHash)
		if err != nil {
			return err
		}

		height := &big.Int{}
		height.SetBytes(snapshotItem.AccountBlockHeight)

		sb.Snapshot[key] = &SnapshotItem{
			AccountBlockHash: &hash,
			AccountBlockHeight: height,
		}
	}

	return nil
}

func (sb *SnapshotBlock) NetDeserialize (buf []byte) error {
	snapshotBlockPB := &vitepb.SnapshotBlockNet{}
	if err := proto.Unmarshal(buf, snapshotBlockPB); err != nil {
		return err
	}

	sb.SetByNetPB(snapshotBlockPB)

	return nil
}

func (sb *SnapshotBlock) NetSerialize () ([]byte, error) {
	return proto.Marshal(sb.GetNetPB())
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
	var genesisSnapshotBlockHash, _ = types.BytesToHash([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	var genesisProducer, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	var genesisSignature = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	var genesisPublicKey = ed25519.PublicKey([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})

	snapshotBLock := &SnapshotBlock{
		Hash: &genesisSnapshotBlockHash,
		PublicKey: genesisPublicKey,
		PrevHash: nil,
		Height: big.NewInt(1),
		Timestamp: uint64(1532084788),
		Producer: &genesisProducer,
		Signature: genesisSignature,
	}
	return snapshotBLock
}

var GenesisSnapshotBlock = GetGenesisSnapshot()