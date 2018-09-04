package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/signer"
	"github.com/vitelabs/go-vite/vite"
	"math/big"
)

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

func NewLedgerApi(vite *vite.Vite) *LedgerApi {
	return &LedgerApi{
		ledgerManager: vite.Ledger(),
		signer:        vite.Signer(),
	}
}

type LedgerApi struct {
	ledgerManager handler_interface.Manager
	signer        *signer.Master
}

func (l LedgerApi) String() string {
	return "LedgerApi"
}

func (l *LedgerApi) CreateTxWithPassphrase(params *SendTxParms) error {
	log.Info("CreateTxWithPassphrase")
	if params == nil {
		return fmt.Errorf("sendTxParms nil")
	}
	if params.Passphrase == "" {
		return fmt.Errorf("sendTxParms Passphrase empty")
	}

	n := new(big.Int)
	amount, ok := n.SetString(params.Amount, 10)
	if !ok {
		return fmt.Errorf("error format of amount")
	}
	b := ledger.AccountBlock{AccountAddress: &params.SelfAddr, To: &params.ToAddr, TokenId: &params.TokenTypeId, Amount: amount}

	// call signer.creattx in order to as soon as possible to send tx
	err := l.signer.CreateTxWithPassphrase(&b, params.Passphrase)

	if err != nil {
		newerr, concerned := TryMakeConcernedError(err)
		if concerned {
			return newerr
		}
		return err
	}

	return nil
}

func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]SimpleBlock, error) {
	log.Info("GetBlocksByAccAddr")

	list, getErr := l.ledgerManager.Ac().GetBlocksByAccAddr(&addr, index, 1, count)

	if getErr != nil {
		log.Info("GetBlocksByAccAddr", "err", getErr)
		if getErr.Code == 1 {
			// todo ask lyd it means no data
			return nil, nil
		}
		return nil, getErr.Err
	}

	simpleBlocks := make([]SimpleBlock, len(list))
	for i, v := range list {

		simpleBlocks[i] = SimpleBlock{
			Timestamp: v.Timestamp,
			Hash:      *v.Hash,
		}

		if v.From != nil {
			simpleBlocks[i].FromAddr = *v.From
		}

		if v.To != nil {
			simpleBlocks[i].ToAddr = *v.To
		}

		if v.Amount != nil {
			simpleBlocks[i].Amount = v.Amount.String()
		}

		if v.Meta != nil {
			simpleBlocks[i].Status = v.Meta.Status
		}

		if v.Balance != nil {
			simpleBlocks[i].Balance = v.Balance.String()
		}

		times := l.getBlockConfirmedTimes(v)
		if times != nil {
			simpleBlocks[i].ConfirmedTimes = times.String()
		}
	}
	return simpleBlocks, nil
}

func (l *LedgerApi) getBlockConfirmedTimes(block *ledger.AccountBlock) *big.Int {
	log.Info("getBlockConfirmedTimes")
	sc := l.ledgerManager.Sc()
	sb, e := sc.GetConfirmBlock(block)
	if e != nil {
		log.Error("GetConfirmBlock ", "err", e)
		return nil
	}
	if sb == nil {
		log.Info("GetConfirmBlock nil")
		return nil
	}

	times, e := sc.GetConfirmTimes(sb)
	if e != nil {
		log.Error("GetConfirmTimes", "err", e)
		return nil
	}

	if times == nil {
		log.Info("GetConfirmTimes nil")
		return nil
	}

	return times
}

func (l *LedgerApi) GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]SimpleBlock, error) {
	log.Info("GetUnconfirmedBlocksByAccAddr")
	return nil, ErrNotSupport
}

func (l *LedgerApi) GetAccountByAccAddr(addr types.Address) (GetAccountResponse, error) {
	log.Info("GetAccountByAccAddr")

	account, err := l.ledgerManager.Ac().GetAccount(&addr)
	if err != nil {
		return GetAccountResponse{}, err
	}

	response := GetAccountResponse{}
	if account == nil {
		log.Error("account == nil")
		return response, nil
	}

	if account.Address != nil {
		response.Addr = *account.Address
	}
	if account.BlockHeight != nil {
		response.BlockHeight = account.BlockHeight.String()
	}

	if len(account.TokenInfoList) != 0 {
		var bs []BalanceInfo
		bs = make([]BalanceInfo, len(account.TokenInfoList))
		for i, v := range account.TokenInfoList {
			amount := "0"
			if v.TotalAmount != nil {
				amount = v.TotalAmount.String()
			}
			bs[i] = BalanceInfo{
				TokenSymbol: v.Token.Symbol,
				TokenName:   v.Token.Name,
				TokenTypeId: *v.Token.Id,
				Balance:     amount,
			}
		}

		response.BalanceInfos = bs
	}
	return response, nil
}

func (l *LedgerApi) GetUnconfirmedInfo(addr types.Address) (GetUnconfirmedInfoResponse, error) {
	log.Info("GetUnconfirmedInfo")

	account, e := l.ledgerManager.Ac().GetUnconfirmedAccount(&addr)
	if e != nil {
		log.Error(e.Error())
		return GetUnconfirmedInfoResponse{}, e
	}

	response := GetUnconfirmedInfoResponse{}

	if account == nil {
		log.Error("account == nil")
		return response, nil
	}

	if account.Address != nil {
		response.Addr = *account.Address
	}
	if account.TotalNumber != nil {
		response.UnConfirmedBlocksLen = account.TotalNumber.String()
	}

	if len(account.TokenInfoList) != 0 {
		blances := make([]BalanceInfo, len(account.TokenInfoList))
		for k, v := range account.TokenInfoList {
			blances[k] = BalanceInfo{
				TokenSymbol: v.Token.Symbol,
				TokenName:   v.Token.Name,
				TokenTypeId: *v.Token.Id,
				Balance:     v.TotalAmount.String(),
			}
		}
		response.BalanceInfos = blances

	}

	return response, nil

}

func (l *LedgerApi) GetInitSyncInfo() (InitSyncResponse, error) {
	log.Info("GetInitSyncInfo")
	i := l.ledgerManager.Sc().GetFirstSyncInfo()

	r := InitSyncResponse{
		StartHeight:      i.BeginHeight.String(),
		TargetHeight:     i.TargetHeight.String(),
		CurrentHeight:    i.CurrentHeight.String(),
		IsFirstSyncDone:  i.IsFirstSyncDone,
		IsStartFirstSync: i.IsFirstSyncStart,
	}

	return r, nil
}

func (l *LedgerApi) GetSnapshotChainHeight() (string, error) {
	log.Info("GetSnapshotChainHeight")
	block, e := l.ledgerManager.Sc().GetLatestBlock()
	if e != nil {
		log.Error(e.Error())
		return "", e
	}
	if block != nil && block.Height != nil {
		return block.Height.String(), nil
	}
	return "", nil
}

func (l *LedgerApi) StartAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

func (l *LedgerApi) StopAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

type PublicTxApi struct {
	txApi *LedgerApi
}



