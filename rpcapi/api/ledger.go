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

func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]ledger.AccountBlock, error) {
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

	simpleBlocks := make([]ledger.AccountBlock, len(list))
	for i, v := range list {
		block := accountBlockToSimpleBlock(v)

		times := l.getBlockConfirmedTimes(v)
		if times != nil {
			//block.ConfirmedTimes = times.String()
		}
		simpleBlocks[i] = *block
	}
	return simpleBlocks, nil
}

func accountBlockToSimpleBlock(v *ledger.AccountBlock) *ledger.AccountBlock {

	if v.AccountAddress != nil {

	}

	//var a = AccountBlock{
	//	Meta: AccountBlockMeta{
	//		AccountId:     "",
	//		Height:        "",
	//		Status:        0,
	//		IsSnapshotted: false,
	//	},
	//	AccountAddress:         types.Address{},
	//	PublicKey:              "",
	//	To:                     types.Address{},
	//	From:                   types.Address{},
	//	FromHash:               types.Hash{},
	//	PrevHash:               types.Hash{},
	//	Hash:                   types.Hash{},
	//	Balance:                "",
	//	Amount:                 "",
	//	Timestamp:              0,
	//	TokenId:                types.TokenTypeId{},
	//	LastBlockHeightInToken: "",
	//	Data:                   "",
	//	SnapshotTimestamp:      types.Hash{},
	//	Signature:              nil,
	//	Nonce:                  nil,
	//	Difficulty:             nil,
	//	FAmount:                "",
	//	ConfirmedTimes:         "",
	//}

	simpleBlock := ledger.AccountBlock{
		Timestamp: v.Timestamp,
		Hash:      v.Hash,
	}
	//if v.From != nil {
	//	simpleBlock.From = *v.From
	//}
	//if v.To != nil {
	//	simpleBlock.To = *v.To
	//}
	//if v.Amount != nil {
	//	simpleBlock.Amount = v.Amount.String()
	//}
	//if v.Meta != nil {
	//	simpleBlock.Meta.Status = v.Meta.Status
	//}
	//if v.Balance != nil {
	//	simpleBlock.Balance = v.Balance.String()
	//}
	return &simpleBlock
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

func (l *LedgerApi) GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]ledger.AccountBlock, error) {
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

func (l *LedgerApi) GetLatestBlock(addr types.Address) (ledger.AccountBlock, error) {
	log.Info("GetLatestBlock")
	b, getError := l.ledgerManager.Ac().GetLatestBlock(&addr)
	if getError != nil {
		return ledger.AccountBlock{}, getError.Err
	}
	return *accountBlockToSimpleBlock(b), nil
}

func (l *LedgerApi) CreateTx(block *ledger.AccountBlock) error {
	return nil
}

//func (l *LedgerApi) StartAutoConfirmTx(addr []string, reply *string) error {
//	return nil
//}
//
//func (l *LedgerApi) StopAutoConfirmTx(addr []string, reply *string) error {
//	return nil
//}

type PublicTxApi struct {
	txApi *LedgerApi
}
