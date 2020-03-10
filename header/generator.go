package header

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

// GenResult represents the result of a block being validated by vm.
type GenResult struct {
	VMBlock *vm_db.VmAccountBlock
	IsRetry bool
	Err     error
}

// IncomingMessage carries the necessary transaction info.
type IncomingMessage struct {
	BlockType byte

	AccountAddress types.Address
	ToAddress      *types.Address
	FromBlockHash  *types.Hash

	TokenId *types.TokenTypeId
	Amount  *big.Int
	Fee     *big.Int
	Data    []byte

	Difficulty *big.Int
}

// SignFunc is the function type defining the callback when a block requires a
// method to sign the transaction in generator.
type SignFunc func(addr types.Address, data []byte) (signedData, pubkey []byte, err error)

type Generator interface {
	GenerateWithBlock(block *ledger.AccountBlock, fromBlock *ledger.AccountBlock) (*GenResult, error)
	GenerateWithMessage(message *IncomingMessage, producer *types.Address, signFunc SignFunc) (*GenResult, error)
	GenerateWithOnRoad(sendBlock *ledger.AccountBlock, producer *types.Address, signFunc SignFunc, difficulty *big.Int) (*GenResult, error)
	GetVMDB() vm_db.VmDb
}
