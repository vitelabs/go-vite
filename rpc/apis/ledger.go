package apis

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/vite"
	"math/big"
)

// !!! Block = Transaction = TX

// Send tx parms
type SendTxParms struct {
	SelfAddr    string // who sends the tx
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

type SimpleBlock struct {
	Timestamp uint64
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

type LedgerApi interface {
	// it will block until the tx is written into the db and broadcast to network. so when the func returns no values
	// that means it has succeed
	CreateTxWithPassphrase(params *SendTxParms, reply *string) error
	// get blocks by page the reply value is []SimpleBlock
	GetBlocksByAccAddr(params *GetBlocksParams, reply *string) error
	// get unconfirmed blocks by page the reply value is []SimpleBlock
	GetUnconfirmedBlocksByAccAddr(params *GetBlocksParams, reply *string) error
	// get account info now it mainly returns balance information, the reply is GetAccountResponse
	GetAccountByAccAddr(addr []string, reply *string) error
	// GetUnconfirmedInfo the reply is GetUnconfirmedInfoResponse
	GetUnconfirmedInfo(addr []string, reply *string) error
	// Get the realtime sync info. the reply is InitSyncResponse
	GetInitSyncInfo(noop interface{}, reply *string) error

	//StartAutoConfirmTx(addr []string, reply *string) error
	//StopAutoConfirmTx(addr []string, reply *string) error
}

func NewLedgerApi(vite *vite.Vite) LedgerApi {
	return &LegerApiImpl{
		ledgerManager: vite.Ledger(),
		vite:          vite,
	}
}

type LegerApiImpl struct {
	ledgerManager handler_interface.Manager
	vite          *vite.Vite
}

func (l *LegerApiImpl) CreateTxWithPassphrase(params *SendTxParms, reply *string) error {
	log.Debug("CreateTxWithPassphrase")
	if params == nil {
		return fmt.Errorf("sendTxParms nil")
	}
	selfaddr, err := types.HexToAddress(params.SelfAddr)
	if err != nil {
		return err
	}
	toaddr, err := types.HexToAddress(params.ToAddr)
	if err != nil {
		return err
	}
	tti, err := types.HexToTokenTypeId(params.TokenTypeId)
	if err != nil {
		return err
	}
	n := new(big.Int)
	amount, ok := n.SetString(params.Amount, 10)
	if !ok {
		return fmt.Errorf("error format of amount")
	}
	b := ledger.AccountBlock{AccountAddress: &selfaddr, To: &toaddr, TokenId: &tti, Amount: amount}

	// call signer.creattx in order to as soon as possible to send tx
	err = l.vite.Signer().CreateTxWithPassphrase(&b, params.Passphrase)

	if err != nil {
		return err
	}
	*reply = "success"
	return nil
}

func (l *LegerApiImpl) GetBlocksByAccAddr(params *GetBlocksParams, reply *string) error {
	log.Debug("GetBlocksByAccAddr")
	if params == nil {
		return fmt.Errorf("sendTxParms nil")
	}
	addr, err := types.HexToAddress(params.Addr)
	if err != nil {
		return err
	}
	list, err := l.ledgerManager.Ac().GetBlocksByAccAddr(&addr, params.index, 1, params.count)
	if err != nil {
		return err
	}
	jsonBlocks := make([]SimpleBlock, len(list))
	for i, v := range list {
		jsonBlocks[i] = SimpleBlock{
			Timestamp: v.Timestamp,
			Amount:    v.Amount.String(),
			FromAddr:  v.From.String(),
			ToAddr:    v.To.String(),
			Status:    v.Meta.Status,
			Hash:      v.Hash.String(),
		}
	}
	return easyJsonReturn(jsonBlocks, reply)
}

func (l *LegerApiImpl) GetUnconfirmedBlocksByAccAddr(params *GetBlocksParams, reply *string) error {
	return nil
}

func (l *LegerApiImpl) GetAccountByAccAddr(addrs []string, reply *string) error {
	log.Debug("GetAccountByAccAddr")
	if len(addrs) != 1 {
		return fmt.Errorf("error length addrs %v", len(addrs))
	}

	addr, err := types.HexToAddress(addrs[0])
	if err != nil {
		return err
	}
	account, err := l.ledgerManager.Ac().GetAccountByAccAddr(&addr)
	if err != nil {
		return err
	}

	var bs []BalanceInfo
	if len(account.TokenList) == 0 {
		bs = nil
	} else {
		bs = make([]BalanceInfo, len(account.TokenList))
		for i, v := range account.TokenList {
			bs[i] = BalanceInfo{
				TokenSymbol: "",
				TokenName:   "",
				TokenTypeId: v.TokenId.String(),
				Balance:     "",
			}
		}
	}

	res := GetAccountResponse{
		Addr:         types.PubkeyToAddress(account.PublicKey[:]).String(),
		BalanceInfos: bs,
		BlockHeight:  "",
	}

	return easyJsonReturn(res, reply)
}

func (l *LegerApiImpl) GetUnconfirmedInfo(addr []string, reply *string) error {
	return nil
}

func (l *LegerApiImpl) GetInitSyncInfo(noop interface{}, reply *string) error {
	log.Debug("GetInitSyncInfo")
	i := l.ledgerManager.Sc().GetFirstSyncInfo()
	r := InitSyncResponse{
		StartHeight:   i.BeginHeight.String(),
		TargetHeight:  i.TargetHeight.String(),
		CurrentHeight: i.CurrentHeight.String(),
	}

	return easyJsonReturn(r, reply)
}

func (l *LegerApiImpl) StartAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

func (l *LegerApiImpl) StopAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

func easyJsonReturn(v interface{}, reply *string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*reply = string(b)
	return nil
}
