package api

import "github.com/vitelabs/go-vite/common/types"

// !!! Block = Transaction = TX

// Send tx parms
type SendTxParms struct {
	SelfAddr    types.Address     // who sends the tx
	ToAddr      types.Address     // who receives the tx
	TokenTypeId types.TokenTypeId // which token will be sent
	Passphrase  string            // sender`s passphrase
	Amount      string            // the amount of specific token will be sent. bigInt
}

type SimpleBlock struct {
	Timestamp      uint64
	Amount         string        // the amount of a specific token had been sent in this block.  bigInt
	FromAddr       types.Address // who sends the tx
	ToAddr         types.Address // who receives the tx
	Status         int           // 0 means unknow, 1 means open (unconfirmed), 2 means closed(already confirmed)
	Hash           types.Hash    // bigInt. the blocks hash
	Balance        string        // current balance
	ConfirmedTimes string        // block`s confirmed times
}

type BalanceInfo struct {
	TokenSymbol string // token symbol example  1200 (symbol)
	TokenName   string // token name
	TokenTypeId types.TokenTypeId
	Balance     string
}

type GetAccountResponse struct {
	Addr         types.Address // Account address
	BalanceInfos []BalanceInfo // Account Balance Infos
	BlockHeight  string        // Account BlockHeight also represents all blocks belong to the account. bigInt.
}

type GetUnconfirmedInfoResponse struct {
	Addr                 types.Address // Account address
	BalanceInfos         []BalanceInfo // Account unconfirmed BalanceInfos (In-transit money)
	UnConfirmedBlocksLen string        // the length of unconfirmed blocks. bigInt
}

type InitSyncResponse struct {
	StartHeight      string // bigInt. where we start sync
	TargetHeight     string // bigInt. when CurrentHeight == TargetHeight means that sync complete
	CurrentHeight    string // bigInt.
	IsFirstSyncDone  bool   // true means sync complete
	IsStartFirstSync bool   // true means sync start
}

type LedgerApi interface {
	// it will block until the tx is written into the db and broadcast to network. so when the func returns no values
	// that means it has succeed
	CreateTxWithPassphrase(params *SendTxParms) error
	// get blocks by page the reply value is []SimpleBlock
	GetBlocksByAccAddr(addr types.Address, index int, count int) ([]SimpleBlock, error)
	// get unconfirmed blocks by page the reply value is []SimpleBlock
	GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]SimpleBlock, error)
	// get account info now it mainly returns balance information, the reply is GetAccountResponse
	GetAccountByAccAddr(addr types.Address) (GetAccountResponse, error)
	// GetUnconfirmedInfo the reply is GetUnconfirmedInfoResponse
	GetUnconfirmedInfo(addr types.Address) (GetUnconfirmedInfoResponse, error)
	// Get the realtime sync info. the reply is InitSyncResponse
	GetInitSyncInfo() (InitSyncResponse, error)
	// GetLatestSnapshotBlock. the reply is the height of Snapshotchain
	GetSnapshotChainHeight() (string, error)
	//StartAutoConfirmTx(addr []string, reply *string) error
	//StopAutoConfirmTx(addr []string, reply *string) error
}
