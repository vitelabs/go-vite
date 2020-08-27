package header

import (
	"math/big"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/vm_db"
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

type Generator interface {
	GenerateWithBlock(block *core.AccountBlock, fromBlock *core.AccountBlock) (*GenResult, error)
	GenerateWithMessage(message *IncomingMessage, producer *types.Address, signFunc interfaces.SignFunc) (*GenResult, error)
	GenerateWithOnRoad(sendBlock *core.AccountBlock, producer *types.Address, signFunc interfaces.SignFunc, difficulty *big.Int) (*GenResult, error)
	GetVMDB() vm_db.VmDb
}
