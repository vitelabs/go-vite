package api

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type AccountBlock struct {
	*RawTxBlock

	TokenInfo *RpcTokenInfo `json:"tokenInfo"`

	ConfirmedTimes *string     `json:"confirmedTimes"`
	ConfirmedHash  *types.Hash `json:"confirmedHash"`

	ReceiveBlockHeight *string     `json:"receiveBlockHeight"`
	ReceiveBlockHash   *types.Hash `json:"receiveBlockHash"`

	Timestamp int64 `json:"timestamp"`
}

type RawTxBlock struct {
	BlockType byte       `json:"blockType"`
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`
	PublicKey      []byte        `json:"publicKey"`

	FromAddress   types.Address     `json:"fromAddress"`
	ToAddress     types.Address     `json:"toAddress""`
	FromBlockHash types.Hash        `json:"fromBlockHash"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	Amount        *string           `json:"amount"`
	Fee           *string           `json:"fee"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`

	Quota         *string       `json:"quota"`
	QuotaUsed     *string       `json:"quotaUsed"`
	LogHash       *types.Hash   `json:"logHash"`
	SendBlockList []*RawTxBlock `json:"sendBlockList"`
}

func (block *RawTxBlock) RpcToLedgerBlock() (*ledger.AccountBlock, error) {
	return copyRawTxToLedgerBlock(block)
}

func ledgerToRpcBlock(chain chain.Chain, lAb *ledger.AccountBlock) (*AccountBlock, error) {
	rpcBlock := &AccountBlock{}

	rawTxBlock, err := copyLedgerBlockToRawTx(chain, lAb)
	if err != nil {
		return nil, err
	}
	rpcBlock.RawTxBlock = rawTxBlock

	// TokenInfo
	if rawTxBlock.TokenId != types.ZERO_TOKENID {
		token, _ := chain.GetTokenInfoById(rawTxBlock.TokenId)
		rpcBlock.TokenInfo = RawTokenInfoToRpc(token, rawTxBlock.TokenId)
	}

	// ReceiveBlockHeight & ReceiveBlockHash
	if lAb.IsSendBlock() {
		receiveBlock, err := chain.GetReceiveAbBySendAb(lAb.Hash)
		if err != nil {
			return nil, err
		}
		if receiveBlock != nil {
			heightStr := strconv.FormatUint(receiveBlock.Height, 10)
			rpcBlock.ReceiveBlockHeight = &heightStr
			rpcBlock.ReceiveBlockHash = &receiveBlock.Hash
		}
	}

	// ConfirmedTimes & ConfirmedHash
	latestSb := chain.GetLatestSnapshotBlock()
	confirmedBlock, err := chain.GetConfirmSnapshotHeaderByAbHash(lAb.Hash)
	if err != nil {
		return nil, err
	}
	if confirmedBlock != nil && latestSb != nil && confirmedBlock.Height <= latestSb.Height {
		confirmedTimeStr := strconv.FormatUint(latestSb.Height-confirmedBlock.Height+1, 10)
		rpcBlock.ConfirmedTimes = &confirmedTimeStr
		rpcBlock.ConfirmedHash = &confirmedBlock.Hash
		rpcBlock.Timestamp = confirmedBlock.Timestamp.Unix()
	}

	return rpcBlock, nil
}

func copyLedgerBlockToRawTx(chain chain.Chain, lAb *ledger.AccountBlock) (*RawTxBlock, error) {
	rawTxBlock := &RawTxBlock{
		BlockType:      lAb.BlockType,
		Hash:           lAb.Hash,
		PrevHash:       lAb.PrevHash,
		AccountAddress: lAb.AccountAddress,
		PublicKey:      lAb.PublicKey,
		FromBlockHash:  lAb.FromBlockHash,
		TokenId:        lAb.TokenId,

		Data:      lAb.Data,
		Signature: lAb.Signature,
		LogHash:   lAb.LogHash,
	}
	//height
	rawTxBlock.Height = strconv.FormatUint(lAb.Height, 10)

	// Quota & QuotaUsed
	quota := strconv.FormatUint(lAb.Quota, 10)
	rawTxBlock.Quota = &quota
	quotaUsed := strconv.FormatUint(lAb.QuotaUsed, 10)
	rawTxBlock.QuotaUsed = &quotaUsed

	// FromAddress & ToAddress
	var amount, fee string
	if lAb.IsSendBlock() {
		rawTxBlock.FromAddress = lAb.AccountAddress
		rawTxBlock.ToAddress = lAb.ToAddress
		if lAb.Amount != nil {
			amount = lAb.Amount.String()
			rawTxBlock.Amount = &amount
		}
		if lAb.Fee != nil {
			fee = lAb.Fee.String()
			rawTxBlock.Fee = &fee
		}
	} else {
		sendBlock, err := chain.GetAccountBlockByHash(lAb.FromBlockHash)
		if err != nil {
			return nil, err
		}
		if sendBlock != nil {
			rawTxBlock.FromAddress = sendBlock.AccountAddress
			rawTxBlock.ToAddress = sendBlock.ToAddress

			rawTxBlock.FromBlockHash = sendBlock.Hash
			rawTxBlock.TokenId = sendBlock.TokenId
			if sendBlock.Amount != nil {
				amount = sendBlock.Amount.String()
				rawTxBlock.Amount = &amount
			}
			if sendBlock.Fee != nil {
				fee = sendBlock.Fee.String()
				rawTxBlock.Fee = &fee
			}
		}
	}

	// Difficulty & Nonce
	rawTxBlock.Nonce = lAb.Nonce
	if lAb.Difficulty != nil {
		difficulty := lAb.Difficulty.String()
		rawTxBlock.Difficulty = &difficulty
	}

	// SendBlockList
	if len(lAb.SendBlockList) > 0 {
		subBlockList := make([]*RawTxBlock, len(lAb.SendBlockList))
		for k, v := range lAb.SendBlockList {
			subRawTx, err := copyLedgerBlockToRawTx(chain, v)
			if err != nil {
				return nil, err
			}
			subBlockList[k] = subRawTx
		}
		rawTxBlock.SendBlockList = subBlockList
	}

	return rawTxBlock, nil
}

func copyRawTxToLedgerBlock(rawTxAb *RawTxBlock) (*ledger.AccountBlock, error) {
	lAb := &ledger.AccountBlock{
		BlockType:      rawTxAb.BlockType,
		Hash:           rawTxAb.Hash,
		PrevHash:       rawTxAb.PrevHash,
		AccountAddress: rawTxAb.AccountAddress,
		PublicKey:      rawTxAb.PublicKey,
		ToAddress:      rawTxAb.ToAddress,

		FromBlockHash: rawTxAb.FromBlockHash,
		TokenId:       rawTxAb.TokenId,

		Data:      rawTxAb.Data,
		Nonce:     rawTxAb.Nonce,
		Signature: rawTxAb.Signature,
	}

	var err error

	lAb.Height, err = strconv.ParseUint(rawTxAb.Height, 10, 64)
	if err != nil {
		return nil, err
	}

	lAb.Amount = big.NewInt(0)
	if rawTxAb.Amount != nil {
		if _, ok := lAb.Amount.SetString(*rawTxAb.Amount, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	lAb.Fee = big.NewInt(0)
	if rawTxAb.Fee != nil {
		if _, ok := lAb.Fee.SetString(*rawTxAb.Fee, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	if rawTxAb.Nonce != nil {
		if rawTxAb.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			difficultyStr, ok := new(big.Int).SetString(*rawTxAb.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = difficultyStr
		}
	}

	if rawTxAb.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*rawTxAb.Quota, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if rawTxAb.QuotaUsed != nil {
		lAb.QuotaUsed, err = strconv.ParseUint(*rawTxAb.QuotaUsed, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if rawTxAb.LogHash != nil {
		lAb.LogHash = rawTxAb.LogHash
	}

	if len(rawTxAb.SendBlockList) > 0 {
		subLAbList := make([]*ledger.AccountBlock, len(rawTxAb.SendBlockList))
		for k, v := range rawTxAb.SendBlockList {
			subLAb, subErr := copyRawTxToLedgerBlock(v)
			if subErr != nil {
				return nil, subErr
			}
			subLAbList[k] = subLAb
		}
		lAb.SendBlockList = subLAbList
	}

	return lAb, nil
}

type RpcAccountInfo struct {
	AccountAddress      types.Address                              `json:"accountAddress"`
	TotalNumber         string                                     `json:"totalNumber"` // uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*RpcTokenBalanceInfo `json:"tokenBalanceInfoMap,omitempty"`
}

type RpcTokenBalanceInfo struct {
	TokenInfo   *RpcTokenInfo `json:"tokenInfo,omitempty"`
	TotalAmount string        `json:"totalAmount"`      // big int
	Number      *string       `json:"number,omitempty"` // uint64
}

type RpcTokenInfo struct {
	TokenName      string            `json:"tokenName"`
	TokenSymbol    string            `json:"tokenSymbol"`
	TotalSupply    *string           `json:"totalSupply,omitempty"` // *big.Int
	Decimals       uint8             `json:"decimals"`
	Owner          types.Address     `json:"owner"`
	PledgeAmount   *string           `json:"pledgeAmount,omitempty"` // *big.Int
	WithdrawHeight string            `json:"withdrawHeight"`         // uint64
	PledgeAddr     types.Address     `json:"pledgeAddr"`
	TokenId        types.TokenTypeId `json:"tokenId"`
	MaxSupply      *string           `json:"maxSupply"` // *big.Int
	OwnerBurnOnly  bool              `json:"ownerBurnOnly"`
	IsReIssuable   bool              `json:"isReIssuable"`
	Index          uint16            `json:"index"`
}

func RawTokenInfoToRpc(tinfo *types.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:      tinfo.TokenName,
			TokenSymbol:    tinfo.TokenSymbol,
			TotalSupply:    nil,
			Decimals:       tinfo.Decimals,
			Owner:          tinfo.Owner,
			PledgeAmount:   nil,
			WithdrawHeight: strconv.FormatUint(tinfo.WithdrawHeight, 10),
			PledgeAddr:     tinfo.PledgeAddr,
			TokenId:        tti,
			OwnerBurnOnly:  tinfo.OwnerBurnOnly,
			IsReIssuable:   tinfo.IsReIssuable,
			Index:          tinfo.Index,
		}
		if tinfo.TotalSupply != nil {
			s := tinfo.TotalSupply.String()
			rt.TotalSupply = &s
		}
		if tinfo.PledgeAmount != nil {
			s := tinfo.PledgeAmount.String()
			rt.PledgeAmount = &s
		}
		if tinfo.MaxSupply != nil {
			s := tinfo.MaxSupply.String()
			rt.MaxSupply = &s
		}
	}
	return rt
}

/*
type AccountBlock struct {
	*ledger.AccountBlock

	FromAddress types.Address `json:"FromAddress"`

	Height string `json:"height"`

	Amount    *string       `json:"amount"`
	Fee       *string       `json:"fee"`
	TokenInfo *RpcTokenInfo `json:"tokenInfo"`

	Difficulty *string `json:"difficulty"`
	Quota      *string `json:"quota"`

	Timestamp int64 `json:"timestamp"`

	ConfirmedTimes *string     `json:"confirmedTimes"`
	ConfirmedHash  *types.Hash `json:"confirmedHash"`

	ReceiveBlockHeight string      `json:"receiveBlockHeight"`
	ReceiveBlockHash   *types.Hash `json:"receiveBlockHash"`
}


// TODO set timestamp
func (ab *AccountBlock) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	lAb := ab.AccountBlock
	if lAb == nil {
		lAb = &ledger.AccountBlock{}
	}

	var err error
	lAb.Height, err = strconv.ParseUint(ab.Height, 10, 64)
	if err != nil {
		return nil, err
	}
	if ab.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*ab.Quota, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	lAb.Amount = big.NewInt(0)
	if ab.Amount != nil {
		if _, ok := lAb.Amount.SetString(*ab.Amount, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	lAb.Fee = big.NewInt(0)
	if ab.Fee != nil {
		if _, ok := lAb.Fee.SetString(*ab.Fee, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	if ab.AccountBlock != nil && ab.AccountBlock.Nonce != nil {
		if ab.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			setString, ok := new(big.Int).SetString(*ab.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = setString
		}
	}

	return lAb, nil
}

func ledgerToRpcBlock(block *ledger.AccountBlock, chain chain.Chain) (*AccountBlock, error) {
	latestSb := chain.GetLatestSnapshotBlock()

	confirmedBlock, err := chain.GetConfirmSnapshotHeaderByAbHash(block.Hash)

	if err != nil {
		return nil, err
	}

	var confirmedTimes uint64
	var confirmedHash types.Hash
	var timestamp int64

	if confirmedBlock != nil && latestSb != nil && confirmedBlock.Height <= latestSb.Height {
		confirmedHash = confirmedBlock.Hash
		confirmedTimes = latestSb.Height - confirmedBlock.Height + 1
		timestamp = confirmedBlock.Timestamp.Unix()
	}

	var fromAddress, toAddress types.Address
	if block.IsReceiveBlock() {
		toAddress = block.AccountAddress
		sendBlock, err := chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return nil, err
		}

		if sendBlock != nil {
			fromAddress = sendBlock.AccountAddress
			block.TokenId = sendBlock.TokenId
			block.Amount = sendBlock.Amount
			block.Fee = sendBlock.Fee
		}
	} else {
		fromAddress = block.AccountAddress
		toAddress = block.ToAddress
	}

	token, _ := chain.GetTokenInfoById(block.TokenId)
	rpcAccountBlock := createAccountBlock(block, token, confirmedTimes)

	if confirmedTimes > 0 {
		rpcAccountBlock.ConfirmedHash = &confirmedHash
	}

	rpcAccountBlock.FromAddress = fromAddress
	rpcAccountBlock.ToAddress = toAddress
	rpcAccountBlock.Timestamp = timestamp

	if block.IsSendBlock() {
		receiveBlock, err := chain.GetReceiveAbBySendAb(block.Hash)
		if err != nil {
			return nil, err
		}
		if receiveBlock != nil {
			rpcAccountBlock.ReceiveBlockHeight = strconv.FormatUint(receiveBlock.Height, 10)
			rpcAccountBlock.ReceiveBlockHash = &receiveBlock.Hash
		}
	}

	return rpcAccountBlock, nil
}

func createAccountBlock(ledgerBlock *ledger.AccountBlock, token *types.TokenInfo, confirmedTimes uint64) *AccountBlock {
	zero := "0"
	quota := strconv.FormatUint(ledgerBlock.Quota, 10)
	confirmedTimeStr := strconv.FormatUint(confirmedTimes, 10)
	ab := &AccountBlock{
		AccountBlock: ledgerBlock,

		Height: strconv.FormatUint(ledgerBlock.Height, 10),
		Quota:  &quota,

		Amount:         &zero,
		Fee:            &zero,
		TokenInfo:      RawTokenInfoToRpc(token, ledgerBlock.TokenId),
		ConfirmedTimes: &confirmedTimeStr,
	}

	if ledgerBlock.Difficulty != nil {
		difficulty := ledgerBlock.Difficulty.String()
		ab.Difficulty = &difficulty
	}

	//if ledgerBlock.Timestamp != nil {
	//	ab.Timestamp = ledgerBlock.Timestamp.Unix()
	//}

	if token != nil {
		ab.TokenInfo = RawTokenInfoToRpc(token, ledgerBlock.TokenId)
	}
	if ledgerBlock.Amount != nil {
		a := ledgerBlock.Amount.String()
		ab.Amount = &a
	}
	if ledgerBlock.Fee != nil {
		s := ledgerBlock.Fee.String()
		ab.Fee = &s
	}

	return ab
}
*/
