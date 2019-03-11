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
	addr, err := types.BytesToAddress(buf[:types.AddressSize+1])
	if err != nil {
		return nil, nil, err
	}
	// Hash
	blockHash, err := types.BytesToHash(buf[types.AddressSize+1 : types.AddressSize+types.HashSize+1])
	if err != nil {
		return nil, nil, err
	}

	height := binary.BigEndian.Uint64(buf[types.AddressSize+types.HashSize+1:])

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
		return errors.New(fmt.Sprintf("The length of pb is %d, %d % %d != 0", lenPb, lenPb, ScItemBytesLen))
	}
	currentPointer := 0
	for currentPointer < lenPb {
		nextPointer := currentPointer + ScItemBytesLen + 1
		if addr, hashHeight, err := bytesToScItem(pb[currentPointer:nextPointer]); err != nil {
			return err
		} else {
			sc[*addr] = hashHeight
		}
		currentPointer = nextPointer
	}

	return nil
}

//func (sc SnapshotContent) proto() map[string]*vitepb.SnapshotItem {
//	pb := make(map[string]*vitepb.SnapshotItem, len(sc))
//
//	for addr, snapshotItem := range sc {
//		pb[hex.EncodeToString(addr.Bytes())] = &vitepb.SnapshotItem{
//			AccountBlockHash:   snapshotItem.Hash.Bytes(),
//			AccountBlockHeight: snapshotItem.Height,
//		}
//	}
//	return pb
//}
//
//func (sc SnapshotContent) deProto(pb map[string]*vitepb.SnapshotItem) error {
//
//	for addrHexString, snapshotItem := range pb {
//		addrBytes, err := hex.DecodeString(addrHexString)
//		if err != nil {
//			return err
//		}
//
//		addr, err := types.BytesToAddress(addrBytes)
//		if err != nil {
//			return err
//		}
//
//		hashHeight := &HashHeight{
//			Height: snapshotItem.AccountBlockHeight,
//		}
//
//		hashHeight.Hash, err = types.BytesToHash(snapshotItem.AccountBlockHash)
//		if err != nil {
//			return err
//		}
//
//		sc[addr] = hashHeight
//	}
//	return nil
//}

type PMSnapshotBlock struct {
	Hash types.Hash `json:"hash"`

	PrevHash types.Hash `json:"prevHash"` // 1
	Height   uint64     `json:"height"`   // 2

	producer  *types.Address    // for cache
	PublicKey ed25519.PublicKey `json:"publicKey"`

	Signature []byte `json:"signature"`

	Timestamp *time.Time `json:"timestamp"` // 3

	StateHash types.Hash `json:"stateHash"` // 4

	SnapshotContent SnapshotContent `json:"snapshotContent"` // 5
}

func (sb *PMSnapshotBlock) hashSourceLength() int {
	// 1 , 2, 3, 4
	size := types.HashSize + 8 + 8 + types.HashSize

	// 5
	size += len(sb.SnapshotContent) * ScItemBytesLen

	// forkName
	forkName := fork.GetRecentForkName(sb.Height)
	if forkName != "" {
		size += len(forkName)
	}

	return size
}

func (sb *PMSnapshotBlock) ComputeHash() types.Hash {
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

	// StateHash
	source = append(source, sb.StateHash.Bytes()...)

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

func (sb *PMSnapshotBlock) Producer() types.Address {
	if sb.producer == nil {
		producer := types.PubkeyToAddress(sb.PublicKey)
		sb.producer = &producer
	}
	return *sb.producer
}

func (sb *PMSnapshotBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(sb.PublicKey, sb.Hash.Bytes(), sb.Signature)
	if verifyErr != nil {
		snapshotBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (sb *PMSnapshotBlock) proto() *vitepb.SnapshotBlock {
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
	pb.StateHash = sb.StateHash.Bytes()
	// 8
	pb.SnapshotContent = sb.SnapshotContent.proto()
	return pb
}

func (sb *PMSnapshotBlock) deProto(pb *vitepb.SnapshotBlock) error {
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
	if sb.StateHash, err = types.BytesToHash(pb.StateHash); err != nil {
		return err
	}

	// 8
	if len(pb.SnapshotContent) > 0 {
		sb.SnapshotContent = make(SnapshotContent, len(pb.SnapshotContent)/ScItemBytesLen)

		if err = sb.SnapshotContent.deProto(pb.SnapshotContent); err != nil {
			return err
		}
	}

	return nil
}

func (sb *PMSnapshotBlock) Serialize() ([]byte, error) {
	pb := sb.proto()
	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (sb *PMSnapshotBlock) Deserialize(buf []byte) error {
	pb := &vitepb.SnapshotBlock{}

	if unmarshalErr := proto.Unmarshal(buf, pb); unmarshalErr != nil {
		return unmarshalErr
	}

	sb.deProto(pb)

	return nil
}
