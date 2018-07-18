package ledger

// !!! Block = Transaction = TX

// Send tx parms
type SendTxParms struct {
	FromAddr    string // who sends the tx
	ToAddr      string // who receives the tx
	Passphrase  string // sender`s passphrase
	TokenTypeId string // which token will be sent
	Amount      string // the amount of specific token will be sent. bigInt
}

type GetBlocksParams struct {
	Addr  string // which addrs
	index int    // page index
	count int    // page count
}

type GetBlocksResponse struct {
	Timestamp int64
	Amount    string // the amount of a specific token had been sent in this block.  bigInt
	FromAddr  string // who sends the tx
	ToAddr    string // who receives the tx
	Status    int    // 0 means unknow, 1 means open (unconfirmed), 2 means closed(already confirmed)
	Hash      string // bigInt. the blocks hash
}

type BalanceInfo struct {
	TokenSymbol string // token symbol example  1200 (symbol)
	TokenName   string // token name
	TokenTypeId string
	Balance     string
}

type GetAccountResponse struct {
	Addr         string        // Account address
	BalanceInfos []BalanceInfo // Account Balance Infos
	BlockHeight  string        // Account BlockHeight. bigInt
}

type GetUnconfirmedInfoResponse struct {
	Addr                 string        // Account address
	BalanceInfos         []BalanceInfo // Account unconfirmed BalanceInfos (In-transit money)
	UnConfirmedBlocksLen int           // the length of unconfirmed blocks.
}

type InitSyncResponse struct {
	StartHeight   string // bigInt. where we start sync
	TargetHeight  string // bigInt. when CurrentHeight == TargetHeight means that sync complete
	CurrentHeight string // bigInt.
}
type JsonApi interface {
	// it will block until the tx is written into the db and broadcast to network. so when the func returns no values
	// that means it has succeed
	CreateTxWithPassphrase(params *SendTxParms, reply *string) error
	// get blocks by page the reply value is GetBlocksResponse
	GetBlocksByAccAddr(params *GetBlocksParams, reply *string) error
	// get unconfirmed blocks by page the reply value is GetBlocksResponse
	GetUnconfirmedBlocksByAccAddr(params *GetBlocksParams, reply *string) error
	// get account info now it mainly returns balance information, the reply is GetAccountResponse
	GetAccountByAccAddr(addr []string, reply *string) error
	// GetUnconfirmedInfo the reply is GetUnconfirmedInfoResponse
	GetUnconfirmedInfo(addr []string, reply *string) error
	// Get the realtime sync info. the reply is InitSyncResponse
	GetInitSyncInfo(noop interface{}, reply *string) error

	StartAutoConfirmTx(addr []string, reply *string) error
	StopAutoConfirmTx(addr []string, reply *string) error
}
