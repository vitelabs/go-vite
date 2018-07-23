package handler_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"github.com/vitelabs/go-vite/ledger/handler"
)

type AccountChain interface {
	HandleGetBlocks (msg *protoTypes.GetAccountBlocksMsg, peer *protoTypes.Peer) error
	HandleSendBlocks (msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer) error
	GetAccountByAccAddr (addr *types.Address) (*ledger.AccountMeta, error)
	GetBlocksByAccAddr (addr *types.Address, index, num, count int) (ledger.AccountBlockList, error)
	CreateTx (block *ledger.AccountBlock) (error)
	CreateTxWithPassphrase (block *ledger.AccountBlock, passphrase string) error
	GetHashListByPaging(index int, num int, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error)
	GetUnconfirmedAccount(addr *types.Address) (*handler.UnconfirmedAccount, error)
	AddListener(addr types.Address, change chan<- struct{})
	RemoveListener(addr types.Address)
}