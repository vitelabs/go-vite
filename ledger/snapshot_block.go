package ledger

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"sort"
	"time"
)

var snapshotBlockLog = log15.New("module", "ledger/snapshot_block")

type SnapshotContentBytesList [][]byte

const ScItemBytesLen = types.AddressSize + types.HashSize + 8

func scItemToBytes(addr *types.Address, hashHeight *HashHeight) []byte {
	bytes := make([]byte, 0, ScItemBytesLen)
	// Address
	bytes = append(bytes, addr.Bytes()...)

	// Hash
	bytes = append(bytes, hashHeight.Hash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, hashHeight.Height)
	bytes = append(bytes, heightBytes...)

	return bytes
}

func bytesToScItem(buf []byte) (*types.Address, *HashHeight, error) {

	// Address
	addr, err := types.BytesToAddress(buf[:types.AddressSize])
	if err != nil {
		return nil, nil, err
	}
	// Hash
	blockHash, err := types.BytesToHash(buf[types.AddressSize : types.AddressSize+types.HashSize])
	if err != nil {
		return nil, nil, err
	}

	height := binary.BigEndian.Uint64(buf[types.AddressSize+types.HashSize:])

	return &addr, &HashHeight{
		Height: height,
		Hash:   blockHash,
	}, nil
}

func NewSnapshotContentBytesList(sc SnapshotContent) SnapshotContentBytesList {
	scbList := make([][]byte, 0, len(sc))
	for addr, hashHeight := range sc {
		scbList = append(scbList, scItemToBytes(&addr, hashHeight))
	}

	return scbList
}

func (scbList SnapshotContentBytesList) Len() int { return len(scbList) }
func (scbList SnapshotContentBytesList) Swap(i, j int) {
	scbList[i], scbList[j] = scbList[j], scbList[i]
}
func (scbList SnapshotContentBytesList) Less(i, j int) bool {
	return bytes.Compare(scbList[i], scbList[j]) <= 0
}

type SnapshotContent map[types.Address]*HashHeight

func (sc SnapshotContent) bytesList() [][]byte {
	scbList := NewSnapshotContentBytesList(sc)
	sort.Sort(scbList)
	return scbList
}

func (sc SnapshotContent) proto() []byte {
	scBytes := make([]byte, 0, ScItemBytesLen*len(sc))
	for addr, hashHeight := range sc {
		scBytes = append(scBytes, scItemToBytes(&addr, hashHeight)...)
	}

	return scBytes
}

func (sc SnapshotContent) deProto(pb []byte) error {
	lenPb := len(pb)
	if lenPb%ScItemBytesLen != 0 {
		return errors.New(fmt.Sprintf("The length of pb is %d, %d / %d != 0", lenPb, lenPb, ScItemBytesLen))
	}
	currentPointer := 0
	for currentPointer < lenPb {
		nextPointer := currentPointer + ScItemBytesLen
		if addr, hashHeight, err := bytesToScItem(pb[currentPointer:nextPointer]); err != nil {
			return err
		} else {
			sc[*addr] = hashHeight
		}
		currentPointer = nextPointer
	}

	return nil
}

type SnapshotBlock struct {
	Hash types.Hash `json:"hash"`

	PrevHash types.Hash `json:"prevHash"` // 1
	Height   uint64     `json:"height"`   // 2

	producer  *types.Address    // for cache
	PublicKey ed25519.PublicKey `json:"publicKey"`

	Signature []byte `json:"signature"`

	Timestamp *time.Time `json:"timestamp"` // 3

	Seed     uint64      `json:Seed`     // 4
	SeedHash *types.Hash `json:SeedHash` // 5

	SnapshotContent SnapshotContent `json:"snapshotContent"` // 6
}

func ComputeSeedHash(seed uint64, prevHash types.Hash, timestamp *time.Time) types.Hash {
	source := make([]byte, 0, 8+types.HashSize+8)

	//Seed
	seedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seedBytes, seed)
	source = append(source, seedBytes...)

	// PrevHash
	source = append(source, prevHash.Bytes()...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(timestamp.Unix()))
	source = append(source, unixTimeBytes...)

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return hash
}

func (sb *SnapshotBlock) hashSourceLength() int {
	// 1 , 2, 3, 4, 5
	size := types.HashSize + 8 + 8 + 8 + types.HashSize

	// 6
	size += len(sb.SnapshotContent) * ScItemBytesLen

	// forkName
	forkName := fork.GetRecentForkName(sb.Height)
	if forkName != "" {
		size += len(forkName)
	}

	return size
}

func (sb *SnapshotBlock) ComputeHash() types.Hash {
	source := make([]byte, 0, sb.hashSourceLength())
	// PrevHash
	source = append(source, sb.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, sb.Height)
	source = append(source, heightBytes...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(sb.Timestamp.Unix()))
	source = append(source, unixTimeBytes...)

	// Seed
	seedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seedBytes, sb.Seed)
	source = append(source, seedBytes...)

	// Seed Hash
	if sb.SeedHash != nil {
		source = append(source, sb.SeedHash.Bytes()...)
	} else {
		source = append(source, make([]byte, 32)...)
	}

	// Snapshot Content
	scBytesList := sb.SnapshotContent.bytesList()

	for _, scBytesItem := range scBytesList {
		source = append(source, scBytesItem...)
	}

	// Add fork name
	forkName := fork.GetRecentForkName(sb.Height)
	if forkName != "" {
		source = append(source, []byte(forkName)...)
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
		snapshotBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (sb *SnapshotBlock) Proto() *vitepb.SnapshotBlock {
	pb := &vitepb.SnapshotBlock{}
	// 1
	pb.Hash = sb.Hash.Bytes()
	// 2
	pb.PrevHash = sb.PrevHash.Bytes()
	// 3
	pb.Height = sb.Height
	// 4
	pb.PublicKey = sb.PublicKey
	// 5
	pb.Signature = sb.Signature
	// 6
	pb.Timestamp = sb.Timestamp.UnixNano()
	// 7
	pb.Seed = sb.Seed
	// 8
	if sb.SeedHash != nil {
		pb.SeedHash = sb.SeedHash.Bytes()
	}
	// 9
	pb.SnapshotContent = sb.SnapshotContent.proto()
	return pb
}

func (sb *SnapshotBlock) DeProto(pb *vitepb.SnapshotBlock) error {
	// 1
	var err error
	if sb.Hash, err = types.BytesToHash(pb.Hash); err != nil {
		return err
	}
	// 2
	if sb.PrevHash, err = types.BytesToHash(pb.PrevHash); err != nil {
		return err
	}
	// 3
	sb.Height = pb.Height

	// 4
	sb.PublicKey = pb.PublicKey
	// 5
	sb.Signature = pb.Signature

	// 6
	timestamp := time.Unix(0, pb.Timestamp)
	sb.Timestamp = &timestamp

	// 7
	sb.Seed = pb.Seed
	// 8
	if len(pb.SeedHash) > 0 {
		seedHash, err := types.BytesToHash(pb.SeedHash)
		if err != nil {
			return err
		}
		sb.SeedHash = &seedHash
	}

	// 9
	if len(pb.SnapshotContent) > 0 {
		sb.SnapshotContent = make(SnapshotContent, len(pb.SnapshotContent)/ScItemBytesLen)

		if err = sb.SnapshotContent.deProto(pb.SnapshotContent); err != nil {
			return err
		}
	}

	return nil
}

func (sb *SnapshotBlock) Serialize() ([]byte, error) {
	pb := sb.Proto()
	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (sb *SnapshotBlock) Deserialize(buf []byte) error {
	pb := &vitepb.SnapshotBlock{}

	if unmarshalErr := proto.Unmarshal(buf, pb); unmarshalErr != nil {
		return unmarshalErr
	}

	if err := sb.DeProto(pb); err != nil {
		return err
	}

	return nil
}
