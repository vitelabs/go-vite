package apis

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/rpc/api_interface"
	"github.com/vitelabs/go-vite/signer"
	"github.com/vitelabs/go-vite/vite"
	"math/big"
)

func NewLedgerApi(vite *vite.Vite) api_interface.LedgerApi {
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

func (l *LegerApiImpl) CreateTxWithPassphrase(params *api_interface.SendTxParms, reply *string) error {
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
	err = l.signer.CreateTxWithPassphrase(&b, params.Passphrase)

	if err != nil {
		return tryMakeConcernedError(err, reply)
	}

	*reply = "success"

	return nil
}

func (l *LegerApiImpl) GetBlocksByAccAddr(params *api_interface.GetBlocksParams, reply *string) error {
	log.Debug("GetBlocksByAccAddr")
	if params == nil {
		return fmt.Errorf("sendTxParms nil")
	}
	addr, err := types.HexToAddress(params.Addr)
	if err != nil {
		return err
	}
	list, err := l.ledgerManager.Ac().GetBlocksByAccAddr(&addr, params.Index, 1, params.Count)
	if err != nil {
		return err
	}
	jsonBlocks := make([]api_interface.SimpleBlock, len(list))
	for i, v := range list {
		jsonBlocks[i] = api_interface.SimpleBlock{
			Timestamp: v.Timestamp,
			Hash:      v.Hash.String(),
		}

		if v.From != nil {
			jsonBlocks[i].FromAddr = v.From.String()
		}

		if v.To != nil {
			jsonBlocks[i].ToAddr = v.To.String()
		}

		if v.Amount != nil {
			jsonBlocks[i].Amount = v.Amount.String()
		}

		if v.Meta != nil {
			jsonBlocks[i].Status = v.Meta.Status
		}

		if v.Balance != nil {
			jsonBlocks[i].Balance = v.Balance.String()
		}

	}
	return easyJsonReturn(jsonBlocks, reply)
}

func (l *LegerApiImpl) GetUnconfirmedBlocksByAccAddr(params *api_interface.GetBlocksParams, reply *string) error {
	log.Debug("GetUnconfirmedBlocksByAccAddr")
	*reply = "not support"
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
	account, err := l.ledgerManager.Ac().GetAccount(&addr)
	if err != nil {
		return err
	}

	var bs []api_interface.BalanceInfo
	if len(account.TokenInfoList) == 0 {
		bs = nil
	} else {
		bs = make([]api_interface.BalanceInfo, len(account.TokenInfoList))
		for i, v := range account.TokenInfoList {
			bs[i] = api_interface.BalanceInfo{
				TokenSymbol: v.Token.Symbol,
				TokenName:   v.Token.Name,
				TokenTypeId: v.Token.Id.String(),
				Balance:     v.TotalAmount.String(),
			}
		}
	}

	res := api_interface.GetAccountResponse{
		Addr:         addrs[0],
		BalanceInfos: bs,
		BlockHeight:  account.BlockHeight.String(),
	}

	return easyJsonReturn(res, reply)
}

func (l *LegerApiImpl) GetUnconfirmedInfo(addr []string, reply *string) error {
	log.Debug("GetUnconfirmedInfo")
	if len(addr) != 1 {
		return fmt.Errorf("error length addrs %v", len(addr))
	}

	address, err := types.HexToAddress(addr[0])

	if err != nil {
		return err
	}
	account, e := l.ledgerManager.Ac().GetUnconfirmedAccount(&address)
	if e != nil {
		return e
	}
	if account == nil {
		*reply = ""
		return nil
	}

	if len(account.TokenInfoList) != 0 {
		blances := make([]api_interface.BalanceInfo, len(account.TokenInfoList))
		for k, v := range account.TokenInfoList {
			blances[k] = api_interface.BalanceInfo{
				TokenSymbol: v.Token.Symbol,
				TokenName:   v.Token.Name,
				TokenTypeId: v.Token.Id.Hex(),
				Balance:     v.TotalAmount.String(),
			}
		}

		return easyJsonReturn(api_interface.GetUnconfirmedInfoResponse{
			Addr:                 account.AccountAddress.Hex(),
			BalanceInfos:         blances,
			UnConfirmedBlocksLen: account.TotalNumber.String(),
		}, reply)
	}

	*reply = ""
	return nil

}

func (l *LegerApiImpl) GetInitSyncInfo(noop interface{}, reply *string) error {
	log.Debug("GetInitSyncInfo")
	i := l.ledgerManager.Sc().GetFirstSyncInfo()

	r := api_interface.InitSyncResponse{
		StartHeight:      i.BeginHeight.String(),
		TargetHeight:     i.TargetHeight.String(),
		CurrentHeight:    i.CurrentHeight.String(),
		IsFirstSyncDone:  i.IsFirstSyncDone,
		IsStartFirstSync: i.IsFirstSyncStart,
	}

	return easyJsonReturn(r, reply)
}

func (l *LegerApiImpl) StartAutoConfirmTx(addr []string, reply *string) error {
	return nil
}

func (l *LegerApiImpl) StopAutoConfirmTx(addr []string, reply *string) error {
	return nil
}
