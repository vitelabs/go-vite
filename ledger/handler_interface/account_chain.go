package handler_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
)

type AccountChain interface {
	HandleGetBlocks (msg *protoTypes.GetAccountBlocksMsg, peer *protoTypes.Peer) error
	HandleSendBlocks (msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer) error
	GetAccountByAccAddr (addr *types.Address) (*ledger.AccountMeta, error)
	GetBlocksByAccAddr (addr *types.Address, index, num, count int) (ledger.AccountBlockList, error)
	CreateTx (block *ledger.AccountBlock) (error)
	CreateTxWithPassphrase (block *ledger.AccountBlock, passphrase string) error
	GetUnconfirmedAccountMeta (addr *types.Address) (*ledger.UnconfirmedMeta, error)
	GetUnconfirmedBlocks (index int, num int, count int, addr *types.Address) ([]*ledger.AccountBlock, error)
}