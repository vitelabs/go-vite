package impl

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/signer"
	"github.com/vitelabs/go-vite/vite"
	"math/big"
)

func NewLedgerApi(vite *vite.Vite) api.LedgerApi {
	return &LegerApiImpl{
		ledgerManager: vite.Ledger(),
		signer:        vite.Signer(),
	}
}

type LegerApiImpl struct {
	ledgerManager handler_interface.Manager
	signer        *signer.Master
}

func (l LegerApiImpl) String() string {
	return "LegerApiImpl"
}

func (l *LegerApiImpl) CreateTxWithPassphrase(params *api.SendTxParms) error {
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
		newerr, concerned := api.TryMakeConcernedError(err)
		if concerned {
			return newerr
		}
		return err
	}

	return nil
}

func (l *LegerApiImpl) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]api.SimpleBlock, error) {
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

	simpleBlocks := make([]api.SimpleBlock, len(list))
	for i, v := range list {

		simpleBlocks[i] = api.SimpleBlock{
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

func (l *LegerApiImpl) getBlockConfirmedTimes(block *ledger.AccountBlock) *big.Int {
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

func (l *LegerApiImpl) GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]api.SimpleBlock, error) {
	log.Info("GetUnconfirmedBlocksByAccAddr")
	return nil, api.ErrNotSupport
}

func (l *LegerApiImpl) GetAccountByAccAddr(addr types.Address) (api.GetAccountResponse, error) {
	log.Info("GetAccountByAccAddr")

	account, err := l.ledgerManager.Ac().GetAccount(&addr)
	if err != nil {
		return api.GetAccountResponse{}, err
	}

	response := api.GetAccountResponse{}
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
		var bs []api.BalanceInfo
		bs = make([]api.BalanceInfo, len(account.TokenInfoList))
		for i, v := range account.TokenInfoList {
			amount := "0"
			if v.TotalAmount != nil {
				amount = v.TotalAmount.String()
			}
			bs[i] = api.BalanceInfo{
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

func (l *LegerApiImpl) GetUnconfirmedInfo(addr types.Address) (api.GetUnconfirmedInfoResponse, error) {
	log.Info("GetUnconfirmedInfo")

	account, e := l.ledgerManager.Ac().GetUnconfirmedAccount(&addr)
	if e != nil {
		log.Error(e.Error())
		return api.GetUnconfirmedInfoResponse{}, e
	}

	response := api.GetUnconfirmedInfoResponse{}

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
		blances := make([]api.BalanceInfo, len(account.TokenInfoList))
		for k, v := range account.TokenInfoList {
			blances[k] = api.BalanceInfo{
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

func (l *LegerApiImpl) GetInitSyncInfo() (api.InitSyncResponse, error) {
	log.Info("GetInitSyncInfo")
	i := l.ledgerManager.Sc().GetFirstSyncInfo()

	r := api.InitSyncResponse{
		StartHeight:      i.BeginHeight.String(),
		TargetHeight:     i.TargetHeight.String(),
		CurrentHeight:    i.CurrentHeight.String(),
		IsFirstSyncDone:  i.IsFirstSyncDone,
		IsStartFirstSync: i.IsFirstSyncStart,
	}

	return r, nil
}

func (l *LegerApiImpl) GetSnapshotChainHeight() (string, error) {
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

func (l *LegerApiImpl) StartAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

func (l *LegerApiImpl) StopAutoConfirmTx(addr []string, reply *string) error {
	return nil
}
