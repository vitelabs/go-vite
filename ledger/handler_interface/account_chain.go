package handler_interface

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type AccountChain interface {
	HandleGetBlocks (msg *protocols.GetAccountBlocksMsg, peer *protocols.Peer) error
	HandleSendBlocks (msg protocols.AccountBlocksMsg, peer *protocols.Peer) error
	GetAccountByAccAddr (addr *types.Address) (*ledger.AccountMeta, error)
	GetBlocksByAccAddr (addr *types.Address, index, num, count int) (ledger.AccountBlockList, error)
	CreateTx (block *ledger.AccountBlock) (error)
	CreateTxWithPassphrase (block *ledger.AccountBlock, passphrase string) error
	GetUnconfirmedAccountMeta (addr *types.Address) (*ledger.UnconfirmedMeta, error)
	GetUnconfirmedBlocks (index int, num int, count int, addr *types.Address) ([]*ledger.AccountBlock, error)
}