package ledger

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
)

//var accountBlockLog = log15.New("module", "ledger/account_block")
//
//type AccountBlockMeta struct {
//	// Account id
//	AccountId uint64 `json:"accountId"`
//
//	// Height
//	Height uint64 `json:"height"`
//
//	// Block status, 1 means open, 2 means closed
//	ReceiveBlockHeights []uint64 `json:"receiveBlockHeights"`
//
//	// Height of Snapshot block which confirm this account block
//	SnapshotHeight uint64 `json:"snapshotHeight"`
//
//	// Height of Snapshot block which pointed by this account block
//	RefSnapshotHeight uint64 `json:"refSnapshotHeight"`
//}
//
//func (abm *AccountBlockMeta) Copy() *AccountBlockMeta {
//	newAbm := *abm
//
//	newAbm.ReceiveBlockHeights = make([]uint64, len(abm.ReceiveBlockHeights))
//	copy(newAbm.ReceiveBlockHeights, abm.ReceiveBlockHeights)
//
//	return &newAbm
//}
//
//func (abm *AccountBlockMeta) Proto() *vitepb.AccountBlockMeta {
//	pb := &vitepb.AccountBlockMeta{
//		AccountId:           abm.AccountId,
//		Height:              abm.Height,
//		ReceiveBlockHeights: abm.ReceiveBlockHeights,
//
//		RefSnapshotHeight: abm.RefSnapshotHeight,
//	}
//	return pb
//}
//
//func (abm *AccountBlockMeta) DeProto(pb *vitepb.AccountBlockMeta) {
//	abm.AccountId = pb.AccountId
//	abm.Height = pb.Height
//	abm.ReceiveBlockHeights = pb.ReceiveBlockHeights
//	abm.RefSnapshotHeight = pb.RefSnapshotHeight
//}
//
//func (abm *AccountBlockMeta) Serialize() ([]byte, error) {
//	return proto.Marshal(abm.Proto())
//}
//
//func (abm *AccountBlockMeta) Deserialize(buf []byte) error {
//	pb := &vitepb.AccountBlockMeta{}
//	if err := proto.Unmarshal(buf, pb); err != nil {
//		return err
//	}
//
//	abm.DeProto(pb)
//	return nil
//}

const (
	BlockTypeSendCreate byte = iota + 1
	BlockTypeSendCall
	BlockTypeSendReward
	BlockTypeReceive
	BlockTypeReceiveError
	BlockTypeSendRefund
)

type PMAccountBlock struct {
	BlockType byte       `json:"blockType"` // 1
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"` // 2
	Height    uint64     `json:"height"`   // 3

	AccountAddress types.Address `json:"accountAddress"` // 4

	PublicKey ed25519.PublicKey `json:"publicKey"`
	ToAddress types.Address     `json:"toAddress"` // 5

	Amount  *big.Int          `json:"amount"`  // 6
	TokenId types.TokenTypeId `json:"tokenId"` // 7

	FromBlockHash types.Hash `json:"fromBlockHash"` // 8

	Quota uint64   `json:"quota"`
	Fee   *big.Int `json:"fee"` // 9

	Data []byte `json:"data"` // 10

	StateHash types.Hash `json:"stateHash"`

	LogHash *types.Hash `json:"logHash"` // 11

	Difficulty *big.Int `json:"difficulty"`
	Nonce      []byte   `json:"nonce"` // 12

	SendBlockList []*PMAccountBlock `json:sendBlockList` // 13

	Signature []byte `json:"signature"`
}

func (ab *PMAccountBlock) Copy() *PMAccountBlock {
	newAb := *ab

	if ab.Amount != nil {
		newAb.Amount = new(big.Int).Set(ab.Amount)
	}

	if ab.Fee != nil {
		newAb.Fee = new(big.Int).Set(ab.Fee)
	}

	newAb.Data = make([]byte, len(ab.Data))
	copy(newAb.Data, ab.Data)

	if ab.LogHash != nil {
		logHash := *ab.LogHash
		newAb.LogHash = &logHash
	}

	if ab.Difficulty != nil {
		newAb.Difficulty = new(big.Int).Set(ab.Difficulty)
	}

	if len(ab.Nonce) > 0 {
		newAb.Nonce = make([]byte, len(ab.Nonce))
		copy(newAb.Nonce, ab.Nonce)
	}

	if len(ab.Signature) > 0 {
		newAb.Signature = make([]byte, len(ab.Signature))
		copy(newAb.Signature, ab.Signature)
	}

	for _, sendBlock := range ab.SendBlockList {
		newAb.SendBlockList = append(newAb.SendBlockList, sendBlock.Copy())
	}
	return &newAb
}

const fieldTagSize = 1

func (ab *PMAccountBlock) hashSourceLength() int {
	// 1, 2, 3 , 4
	size := fieldTagSize + 1 + fieldTagSize + types.HashSize + fieldTagSize + 8 + fieldTagSize + types.AddressSize
	if ab.IsSendBlock() {
		// 5, 6, 7
		size += fieldTagSize + types.AddressSize + fieldTagSize + len(ab.Amount.Bytes()) + fieldTagSize + types.TokenTypeIdSize
	} else {
		// 8
		size += fieldTagSize + types.HashSize
	}

	// 9
	size += fieldTagSize
	if ab.Fee != nil {
		size += len(ab.Fee.Bytes())
	}

	// 10
	size += fieldTagSize
	size += len(ab.Data)

	// 11
	size += fieldTagSize
	if ab.LogHash != nil {
		size += len(ab.LogHash)
	}

	// 12
	size += fieldTagSize
	size += len(ab.Nonce)

	// 13
	size += fieldTagSize
	for _, sendBlock := range ab.SendBlockList {
		size += sendBlock.hashSourceLength()
	}

	return size
}

func (ab *PMAccountBlock) hashSource() []byte {
	source := make([]byte, 0, ab.hashSourceLength())
	// BlockType
	source = append(source, byte(1))
	source = append(source, ab.BlockType)

	// PrevHash
	source = append(source, byte(2))
	source = append(source, ab.PrevHash.Bytes()...)

	// Height
	source = append(source, byte(3))
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, ab.Height)
	source = append(source, heightBytes...)

	// AccountAddress
	source = append(source, byte(4))
	source = append(source, ab.AccountAddress.Bytes()...)

	if ab.IsSendBlock() {
		// ToAddress
		source = append(source, byte(5))
		source = append(source, ab.ToAddress.Bytes()...)
		// Amount
		source = append(source, byte(6))
		source = append(source, ab.Amount.Bytes()...)
		// TokenId
		source = append(source, byte(7))
		source = append(source, ab.TokenId.Bytes()...)
	} else {
		// FromBlockHash
		source = append(source, byte(8))
		source = append(source, ab.FromBlockHash.Bytes()...)
	}

	// Fee
	fee := ab.Fee
	if fee == nil {
		fee = big.NewInt(0)
	}
	source = append(source, byte(9))
	source = append(source, fee.Bytes()...)

	// Data
	source = append(source, byte(10))
	source = append(source, ab.Data...)

	// LogHash
	source = append(source, byte(11))
	if ab.LogHash != nil {
		source = append(source, ab.LogHash.Bytes()...)
	}

	// Nonce
	source = append(source, byte(12))
	source = append(source, ab.Nonce...)

	// Send block list
	source = append(source, byte(13))
	for _, sendBlock := range ab.SendBlockList {
		source = append(source, sendBlock.hashSource()...)
	}
	return source
}

func (ab *PMAccountBlock) ComputeHash() types.Hash {
	source := ab.hashSource()

	hash, _ := types.BytesToHash(crypto.Hash256(source))

	return hash
}

func (ab *PMAccountBlock) Producer() types.Address {
	return types.PubkeyToAddress(ab.PublicKey)
}

func (ab *PMAccountBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(ab.PublicKey, ab.Hash.Bytes(), ab.Signature)
	if verifyErr != nil {
		accountBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

//func (ab *AccountBlock) proto() *vitepb.AccountBlock {
//	pb := &vitepb.AccountBlock{}
//	pb.BlockType = uint32(ab.BlockType)
//	pb.Hash = ab.Hash.Bytes()
//	pb.Height = ab.Height
//	pb.PrevHash = ab.PrevHash.Bytes()
//
//	if ab.IsSendBlock() {
//		pb.ToAddress = ab.ToAddress.Bytes()
//		pb.Amount = ab.Amount.Bytes()
//		pb.TokenId = ab.TokenId.Bytes()
//	} else {
//		pb.FromBlockHash = ab.FromBlockHash.Bytes()
//	}
//
//	pb.Quota = ab.Quota
//
//	fee := ab.Fee
//	if fee == nil {
//		fee = big.NewInt(0)
//	}
//	pb.Fee = fee.Bytes()
//
//	pb.SnapshotHash = ab.SnapshotHash.Bytes()
//	pb.Data = ab.Data
//	pb.Timestamp = ab.Timestamp.UnixNano()
//
//	if ab.LogHash != nil {
//		pb.LogHash = ab.LogHash.Bytes()
//	}
//
//	if ab.Difficulty != nil {
//		pb.Difficulty = ab.Difficulty.Bytes()
//		// Difficulty is big.NewInt(0)
//		if len(pb.Difficulty) <= 0 {
//			pb.Difficulty = []byte{0}
//		}
//	}
//	pb.Nonce = ab.Nonce
//	pb.Signature = ab.Signature
//	return pb
//}
//
//func (ab *AccountBlock) DbProto() *vitepb.AccountBlock {
//	pb := ab.proto()
//	pb.StateHash = ab.StateHash.Bytes()
//	if len(ab.Producer()) > 0 && !bytes.Equal(ab.Producer().Bytes(), ab.AccountAddress.Bytes()) {
//		pb.PublicKey = ab.PublicKey
//	}
//
//	return pb
//}
//
//func (ab *AccountBlock) Proto() *vitepb.AccountBlock {
//	pb := ab.proto()
//	pb.AccountAddress = ab.AccountAddress.Bytes()
//	pb.PublicKey = ab.PublicKey
//	return pb
//}
//
//func (ab *AccountBlock) DeProto(pb *vitepb.AccountBlock) {
//	ab.BlockType = byte(pb.BlockType)
//	ab.Hash, _ = types.BytesToHash(pb.Hash)
//	ab.Height = pb.Height
//	ab.PrevHash, _ = types.BytesToHash(pb.PrevHash)
//	ab.AccountAddress, _ = types.BytesToAddress(pb.AccountAddress)
//	ab.PublicKey = pb.PublicKey
//
//	if len(pb.ToAddress) > 0 {
//		ab.ToAddress, _ = types.BytesToAddress(pb.ToAddress)
//	}
//	if len(pb.TokenId) > 0 {
//		ab.TokenId, _ = types.BytesToTokenTypeId(pb.TokenId)
//	}
//
//	ab.Amount = big.NewInt(0)
//
//	if len(pb.Amount) > 0 {
//		ab.Amount.SetBytes(pb.Amount)
//	}
//
//	if len(pb.FromBlockHash) > 0 {
//		ab.FromBlockHash, _ = types.BytesToHash(pb.FromBlockHash)
//	}
//	ab.Quota = pb.Quota
//
//	ab.Fee = big.NewInt(0)
//	if len(pb.Fee) > 0 {
//		ab.Fee.SetBytes(pb.Fee)
//	}
//
//	ab.SnapshotHash, _ = types.BytesToHash(pb.SnapshotHash)
//	ab.Data = pb.Data
//	timestamp := time.Unix(0, pb.Timestamp)
//	ab.Timestamp = &timestamp
//	ab.StateHash, _ = types.BytesToHash(pb.StateHash)
//
//	if len(pb.LogHash) > 0 {
//		logHash, _ := types.BytesToHash(pb.LogHash)
//		ab.LogHash = &logHash
//	}
//
//	if len(pb.Difficulty) > 0 {
//		ab.Difficulty = new(big.Int).SetBytes(pb.Difficulty)
//	}
//	ab.Nonce = pb.Nonce
//	ab.Signature = pb.Signature
//
//}
//
//

//
//func (ab *AccountBlock) DbSerialize() ([]byte, error) {
//	return proto.Marshal(ab.DbProto())
//}
//
//func (ab *AccountBlock) DbDeserialize(buf []byte) error {
//	pb := &vitepb.AccountBlock{}
//	if err := proto.Unmarshal(buf, pb); err != nil {
//		return err
//	}
//	ab.DeProto(pb)
//	return nil
//}
//
//func (ab *AccountBlock) Serialize() ([]byte, error) {
//	return proto.Marshal(ab.Proto())
//}
//
//func (ab *AccountBlock) Deserialize(buf []byte) error {
//	pb := &vitepb.AccountBlock{}
//	if err := proto.Unmarshal(buf, pb); err != nil {
//		return err
//	}
//	ab.DeProto(pb)
//
//	return nil
//}
//
func (ab *PMAccountBlock) IsSendBlock() bool {
	return ab.BlockType == BlockTypeSendCreate ||
		ab.BlockType == BlockTypeSendCall ||
		ab.BlockType == BlockTypeSendReward ||
		ab.BlockType == BlockTypeSendRefund
}

func (ab *PMAccountBlock) IsReceiveBlock() bool {
	return ab.BlockType == BlockTypeReceive || ab.BlockType == BlockTypeReceiveError
}
