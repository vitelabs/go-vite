package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

const (
	BlockTypeSendCreate = iota
	BlockTypeSendCall
	BlockTypeSendMintage
	BlockTypeReceive
	BlockTypeReceiveError
	BlockTypeReceiveReward
)

type CreateAccountBlockFunc func(from, to types.Address, txType, depth uint64) VmAccountBlock

type VmAccountBlock interface {
	// Account block height
	Height() *big.Int
	SetHeight(*big.Int)
	// Receiver account address
	ToAddress() types.Address
	SetToAddress(types.Address)
	// Sender account address
	AccountAddress() types.Address
	SetAccountAddress(types.Address)
	// Sender block hash, exists in receive block
	FromBlockHash() types.Hash
	SetFromBlockHash(types.Hash)
	// Transaction type of current block
	BlockType() uint64
	SetBlockType(uint64)
	// Last block hash
	PrevHash() types.Hash
	SetPrevHash(types.Hash)
	// Amount of this transaction
	Amount() *big.Int
	SetAmount(*big.Int)
	// Id of token received or sent
	TokenId() types.TokenTypeId
	SetTokenId(types.TokenTypeId)
	// Create contract fee of vite token
	CreateFee() *big.Int
	SetCreateFee(*big.Int)
	// Input data
	Data() []byte
	SetData([]byte)
	// State root hash of current account
	StateHash() types.Hash
	SetStateHash(types.Hash)
	// Send block summary hash list
	SendBlockHashList() []types.Hash
	AppendSendBlockHash(types.Hash)
	// Snapshot block hash
	SnapshotHash() types.Hash
	SetSnapshotHash(types.Hash)
	// Call or create depth of current account block
	Depth() uint64
	SetDepth(uint64)
	// Quota used of current block
	Quota() uint64
	SetQuota(uint64)
	// Hash value of Height, AccountAddress, ToAddress, BlockType, Amount, TokenTypeId, Data, Depth
	SummaryHash() types.Hash
}

type VmSnapshotBlock interface {
	Height() *big.Int
	Timestamp() uint64
	Hash() types.Hash
}
