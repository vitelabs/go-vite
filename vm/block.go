package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

const (
	TxTypeSendCreate = iota
	TxTypeSendCall
	TxTypeReceive
	TxTypeReceiveError
)

type CreateBlockFunc func(from, to types.Address, txType, depth uint64) VmBlock

type VmBlock interface {
	// Account block height
	Height() *big.Int
	SetHeight(*big.Int)
	// Receiver account address
	To() types.Address
	SetTo(types.Address)
	// Sender account address
	From() types.Address
	SetFrom(types.Address)
	// Sender block hash, exists in receive block
	FromHash() types.Hash
	SetFromHash(types.Hash)
	// Transaction type of current block
	TxType() uint64
	SetTxType(uint64)
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
	SummaryHashList() []types.Hash
	AppendSummaryHash(types.Hash)
	// Snapshot block hash
	SnapshotHash() types.Hash
	SetSnapshotHash(types.Hash)
	// Call or create depth of current account block
	Depth() uint64
	SetDepth(uint64)
	// Quota used of current block
	Quota() uint64
	SetQuota(uint64)
	// Hash value of Height, From, To, TxType, Amount, TokenTypeId, Data, Depth
	SummaryHash() types.Hash
}
