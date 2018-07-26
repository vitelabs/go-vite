package handler_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
)

type AccountChain interface {
	HandleGetBlocks(msg *protoTypes.GetAccountBlocksMsg, peer *protoTypes.Peer) error
	HandleSendBlocks(msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer) error
	GetAccountByAccAddr(addr *types.Address) (*ledger.AccountMeta, error)
	GetBlocksByAccAddr(addr *types.Address, index, num, count int) (ledger.AccountBlockList, error)
	CreateTx(block *ledger.AccountBlock) error
	CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error
	GetUnconfirmedTxHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error)
	GetUnconfirmedTxHashsByTkId(index, num, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error)
	GetUnconfirmedAccount(addr *types.Address) (*UnconfirmedAccount, error)
	AddListener(addr types.Address, change chan<- struct{})
	RemoveListener(addr types.Address)
	GetAccount(accountAddress *types.Address) (*Account, error)
}

// pack the data for handler
type TokenInfo struct {
	Token       *ledger.Mintage
	TotalAmount *big.Int      // in UnconfirmedAccount is Amount, in Account is balance
}

type UnconfirmedAccount struct {
	AccountAddress *types.Address
	TotalNumber    *big.Int
	TokenInfoList  []*TokenInfo
}

type Account struct {
	AccountAddress *types.Address
	BlockHeight    *big.Int
	TokenInfoList  []*TokenInfo
}
