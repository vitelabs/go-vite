package ledger

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
)

var accountBlockLog = log15.New("module", "ledger/account_block")

const (
	BlockTypeSendCreate   = byte(1)
	BlockTypeSendCall     = byte(2)
	BlockTypeSendReward   = byte(3)
	BlockTypeReceive      = byte(4)
	BlockTypeReceiveError = byte(5)
	BlockTypeSendRefund   = byte(6)

	BlockTypeGenesisReceive = byte(7)
)

type PMAccountBlock struct {
	BlockType byte       `json:"blockType"` // 1
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"` // 2
	Height    uint64     `json:"height"`   // 3

	AccountAddress types.Address `json:"accountAddress"` // 4

	producer  *types.Address    // for cache
	PublicKey ed25519.PublicKey `json:"publicKey"`
	ToAddress types.Address     `json:"toAddress"` // 5

	Amount  *big.Int          `json:"amount"`  // 6
	TokenId types.TokenTypeId `json:"tokenId"` // 7

	FromBlockHash types.Hash `json:"fromBlockHash"` // 8

	Data []byte `json:"data"` // 9

	Quota uint64   `json:"quota"`
	Fee   *big.Int `json:"fee"` // 10

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

func (ab *PMAccountBlock) hashSourceLength() int {
	// 1, 2, 3 , 4
	size := 1 + types.HashSize + 8 + types.AddressSize
	if ab.IsSendBlock() {
		// 5, 6, 7
		size += types.AddressSize + len(ab.Amount.Bytes()) + types.TokenTypeIdSize
	} else {
		// 8
		size += types.HashSize
	}

	// 9
	if len(ab.Data) > 0 {
		size += types.HashSize
	}

	// 10
	if ab.Fee != nil {
		size += len(ab.Fee.Bytes())
	}

	// 11
	if ab.LogHash != nil {
		size += types.HashSize
	}

	// 12
	size += len(ab.Nonce)

	// 13
	for _, sendBlock := range ab.SendBlockList {
		size += sendBlock.hashSourceLength()
	}

	return size
}

func (ab *PMAccountBlock) hashSource() []byte {
	source := make([]byte, 0, ab.hashSourceLength())
	// BlockType
	source = append(source, ab.BlockType)

	// PrevHash
	source = append(source, ab.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, ab.Height)
	source = append(source, heightBytes...)

	// AccountAddress
	source = append(source, ab.AccountAddress.Bytes()...)

	if ab.IsSendBlock() {
		// ToAddress
		source = append(source, ab.ToAddress.Bytes()...)
		// Amount
		source = append(source, ab.Amount.Bytes()...)
		// TokenId
		source = append(source, ab.TokenId.Bytes()...)
	} else {
		// FromBlockHash
		source = append(source, ab.FromBlockHash.Bytes()...)
	}

	// Data
	if len(ab.Data) > 0 {
		dataHashBytes := crypto.Hash256(ab.Data)
		source = append(source, dataHashBytes...)
	}

	// Fee
	if ab.Fee != nil {
		source = append(source, ab.Fee.Bytes()...)
	}

	// LogHash
	if ab.LogHash != nil {
		source = append(source, ab.LogHash.Bytes()...)
	}

	// Nonce
	source = append(source, ab.Nonce...)

	// Send block list
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
	if ab.producer == nil {
		producer := types.PubkeyToAddress(ab.PublicKey)
		ab.producer = &producer
	}

	return *ab.producer
}

func (ab *PMAccountBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(ab.PublicKey, ab.Hash.Bytes(), ab.Signature)
	if verifyErr != nil {
		accountBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (ab *PMAccountBlock) proto() *vitepb.PMAccountBlock {
	pb := &vitepb.PMAccountBlock{}
	// 1
	pb.BlockType = vitepb.PMAccountBlock_BlockType(ab.BlockType)
	// 2
	pb.Hash = ab.Hash.Bytes()
	// 3
	pb.Height = ab.Height
	// 4
	if ab.Height > 1 {
		pb.PrevHash = ab.PrevHash.Bytes()
	}
	// 5
	pb.AccountAddress = ab.AccountAddress.Bytes()
	// 6
	pb.PublicKey = ab.PublicKey
	if ab.IsSendBlock() {
		// 7
		pb.ToAddress = ab.ToAddress.Bytes()
		// 8
		pb.Amount = ab.Amount.Bytes()
		// 9
		pb.TokenId = ab.TokenId.Bytes()
	} else {
		// 10
		pb.FromBlockHash = ab.FromBlockHash.Bytes()
	}

	// 11
	pb.Data = ab.Data

	// 12
	pb.Quota = ab.Quota

	if ab.Fee != nil {
		// 13
		pb.Fee = ab.Fee.Bytes()
	}

	// 14
	pb.StateHash = ab.StateHash.Bytes()

	if ab.LogHash != nil {
		// 15
		pb.LogHash = ab.LogHash.Bytes()
	}

	if ab.Difficulty != nil {
		// 16
		pb.Difficulty = ab.Difficulty.Bytes()
	}
	// 17
	pb.Nonce = ab.Nonce
	// 18
	pb.SendBlockList = make([]*vitepb.PMAccountBlock, 0, len(ab.SendBlockList))
	for _, sendBlock := range ab.SendBlockList {
		pb.SendBlockList = append(pb.SendBlockList, sendBlock.proto())
	}
	// 19
	pb.Signature = ab.Signature
	return pb
}

func (ab *PMAccountBlock) deProto(pb *vitepb.PMAccountBlock) error {
	var err error
	// 1
	ab.BlockType = byte(pb.BlockType)
	// 2
	ab.Hash, _ = types.BytesToHash(pb.Hash)
	// 3
	ab.Height = pb.Height
	// 4
	if ab.Height > 1 {
		ab.PrevHash, _ = types.BytesToHash(pb.PrevHash)
	}
	// 5
	if ab.AccountAddress, err = types.BytesToAddress(pb.AccountAddress); err != nil {
		return err
	}
	// 6
	ab.PublicKey = pb.PublicKey

	if ab.IsSendBlock() {
		// 7
		if ab.ToAddress, err = types.BytesToAddress(pb.ToAddress); err != nil {
			return err
		}

		// 8
		ab.Amount = big.NewInt(0)
		if len(pb.Amount) > 0 {
			ab.Amount.SetBytes(pb.Amount)
		}

		// 9
		if ab.TokenId, err = types.BytesToTokenTypeId(pb.TokenId); err != nil {
			return err
		}
	} else {
		// 10
		if ab.FromBlockHash, err = types.BytesToHash(pb.FromBlockHash); err != nil {
			return err
		}
	}

	// 11
	ab.Data = pb.Data

	// 12
	ab.Quota = pb.Quota

	// 13
	ab.Fee = big.NewInt(0)
	if len(pb.Fee) > 0 {
		ab.Fee.SetBytes(pb.Fee)
	}

	// 14
	if len(pb.StateHash) > 0 {
		if ab.StateHash, err = types.BytesToHash(pb.StateHash); err != nil {
			return err
		}
	}

	// 15
	if len(pb.LogHash) > 0 {
		logHash, err := types.BytesToHash(pb.LogHash)
		if err != nil {
			return err
		}

		ab.LogHash = &logHash
	}

	// 16
	if len(pb.Difficulty) > 0 {
		ab.Difficulty = new(big.Int).SetBytes(pb.Difficulty)
	}
	// 17
	ab.Nonce = pb.Nonce

	// 18
	ab.SendBlockList = make([]*PMAccountBlock, 0, len(pb.SendBlockList))
	for _, pbSendBlock := range pb.SendBlockList {
		sendBlock := &PMAccountBlock{}
		if err := sendBlock.deProto(pbSendBlock); err != nil {
			return err
		}
		ab.SendBlockList = append(ab.SendBlockList, sendBlock)
	}
	// 19
	ab.Signature = pb.Signature
	return nil
}

func (ab *PMAccountBlock) Serialize() ([]byte, error) {
	return proto.Marshal(ab.proto())
}

func (ab *PMAccountBlock) Deserialize(buf []byte) error {
	pb := &vitepb.PMAccountBlock{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	ab.deProto(pb)
	return nil
}

func (ab *PMAccountBlock) IsSendBlock() bool {
	return ab.BlockType == BlockTypeSendCreate ||
		ab.BlockType == BlockTypeSendCall ||
		ab.BlockType == BlockTypeSendReward ||
		ab.BlockType == BlockTypeSendRefund
}

func (ab *PMAccountBlock) IsReceiveBlock() bool {
	return ab.BlockType == BlockTypeReceive ||
		ab.BlockType == BlockTypeReceiveError ||
		ab.BlockType == BlockTypeGenesisReceive
}
