package ledger

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/vitepb"
	"time"
)

type SnapshotContent map[types.Address]*SnapshotContentItem

func (*SnapshotContent) Serialize() ([]byte, error) {
	return nil, nil
}
func (*SnapshotContent) Deserialize([]byte) error {
	return nil
}

func (*SnapshotContent) Hash() []byte {
	return nil
}

type SnapshotContentItem struct {
	AccountBlockHeight uint64
	AccountBlockHash   types.Hash
}

type SnapshotBlock struct {
	Hash     types.Hash
	PrevHash types.Hash
	Height   uint64
	producer *types.Address

	PublicKey ed25519.PublicKey
	Signature []byte

	Timestamp *time.Time

	SnapshotHash    *types.Hash
	SnapshotContent SnapshotContent
}

func (sb *SnapshotBlock) ComputeHash() types.Hash {
	var source []byte
	// PrevHash
	source = append(source, sb.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, sb.Height)
	source = append(source, heightBytes...)

	// PublicKey
	source = append(source, sb.PublicKey...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(sb.Timestamp.Unix()))
	source = append(source, unixTimeBytes...)

	// SnapshotHash
	if sb.SnapshotHash != nil {
		source = append(source, sb.SnapshotHash.Bytes()...)
	}

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return hash
}

func (sb *SnapshotBlock) Producer() types.Address {
	if sb.producer == nil {
		producer := types.PubkeyToAddress(sb.PublicKey)
		sb.producer = &producer
	}
	return *sb.producer
}

func (sb *SnapshotBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(sb.PublicKey, sb.Hash.Bytes(), sb.Signature)
	if verifyErr != nil {
		accountBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (sb *SnapshotBlock) Proto() *vitepb.SnapshotBlock {
	pb := &vitepb.SnapshotBlock{}
	pb.Hash = sb.Hash.Bytes()
	pb.PrevHash = sb.PrevHash.Bytes()
	pb.Height = sb.Height
	pb.PublicKey = sb.PublicKey
	pb.Signature = sb.Signature
	pb.Timestamp = uint64(sb.Timestamp.UnixNano())
	if sb.SnapshotHash != nil {
		pb.SnapshotHash = sb.SnapshotHash.Bytes()
	}

	return pb
}

func (sb *SnapshotBlock) DeProto(pb *vitepb.SnapshotBlock) {
	sb.Hash, _ = types.BytesToHash(pb.Hash)
	sb.PrevHash, _ = types.BytesToHash(pb.PrevHash)
	sb.Height = pb.Height
	sb.PublicKey = pb.PublicKey
	sb.Signature = pb.Signature
	// TODO Timestamp
	//sb.Timestamp = pb.Timestamp

	if len(pb.SnapshotHash) >= 0 {
		snapshotHash, _ := types.BytesToHash(pb.SnapshotHash)
		sb.SnapshotHash = &snapshotHash
	}
}

func (*SnapshotBlock) DbSerialize() ([]byte, error) {

	return nil, nil
}

func (*SnapshotBlock) DbDeserialize([]byte) error {
	return nil
}

func (*SnapshotBlock) Serialize() ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) Deserialize([]byte) error {
	return nil
}

func GetGenesisSnapshotBlock() *SnapshotBlock {
	timestamp := time.Unix(1537361101, 0)
	genesisSnapshotBlock := &SnapshotBlock{
		Height:    1,
		Timestamp: &timestamp,
		PublicKey: GenesisPublicKey,
	}
	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()
	genesisSnapshotBlock.Signature, _ = hex.DecodeString("2147fb12ea96ab8561c02c9333ad4e0afc8420f036107582c269bb7e2ebf16443536996bacebef17455703de8a9a6c95998ed3fb3a7a4f44adb0c196572fb20b")

	return genesisSnapshotBlock
}
