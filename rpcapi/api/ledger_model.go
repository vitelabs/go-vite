package api

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotChunk struct {
	AccountBlocks []*ledger.AccountBlock
	SnapshotBlock *SnapshotBlock
}

type AccountBlock struct {
	BlockType byte       `json:"blockType"`
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`
	PublicKey      []byte        `json:"publicKey"`

	Producer types.Address `json:"producer"`

	FromAddress   types.Address     `json:"fromAddress"`
	ToAddress     types.Address     `json:"toAddress"`
	FromBlockHash types.Hash        `json:"fromBlockHash"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	Amount        *string           `json:"amount"`
	Fee           *string           `json:"fee"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`

	Quota         *string         `json:"quota"`
	QuotaUsed     *string         `json:"quotaUsed"`
	LogHash       *types.Hash     `json:"logHash"`
	SendBlockList []*AccountBlock `json:"sendBlockList"`

	// extra info below
	TokenInfo *RpcTokenInfo `json:"tokenInfo"`

	ConfirmedTimes *string     `json:"confirmedTimes"`
	ConfirmedHash  *types.Hash `json:"confirmedHash"`

	ReceiveBlockHeight *string     `json:"receiveBlockHeight"`
	ReceiveBlockHash   *types.Hash `json:"receiveBlockHash"`

	Timestamp int64 `json:"timestamp"`
}

type SnapshotBlock struct {
	Producer types.Address `json:"producer"`
	*ledger.SnapshotBlock
	Timestamp int64 `json:"timestamp"`
}

func (block *AccountBlock) RpcToLedgerBlock() (*ledger.AccountBlock, error) {
	lAb := &ledger.AccountBlock{
		BlockType:      block.BlockType,
		Hash:           block.Hash,
		PrevHash:       block.PrevHash,
		AccountAddress: block.AccountAddress,
		PublicKey:      block.PublicKey,
		ToAddress:      block.ToAddress,

		FromBlockHash: block.FromBlockHash,
		TokenId:       block.TokenId,

		Data:      block.Data,
		Nonce:     block.Nonce,
		Signature: block.Signature,
		LogHash:   block.LogHash,
	}

	var err error

	lAb.Height, err = strconv.ParseUint(block.Height, 10, 64)
	if err != nil {
		return nil, err
	}

	lAb.Amount = big.NewInt(0)
	if block.Amount != nil {
		if _, ok := lAb.Amount.SetString(*block.Amount, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	lAb.Fee = big.NewInt(0)
	if block.Fee != nil {
		if _, ok := lAb.Fee.SetString(*block.Fee, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	if block.Nonce != nil {
		if block.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			difficultyStr, ok := new(big.Int).SetString(*block.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = difficultyStr
		}
	}

	if block.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*block.Quota, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if block.QuotaUsed != nil {
		lAb.QuotaUsed, err = strconv.ParseUint(*block.QuotaUsed, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if !types.IsContractAddr(block.AccountAddress) {
		return lAb, nil
	}

	if len(block.SendBlockList) > 0 {
		subLAbList := make([]*ledger.AccountBlock, len(block.SendBlockList))
		for k, v := range block.SendBlockList {
			subLAb, subErr := v.RpcToLedgerBlock()
			if subErr != nil {
				return nil, subErr
			}
			subLAbList[k] = subLAb
		}
		lAb.SendBlockList = subLAbList
	}

	return lAb, nil
}

func (block *AccountBlock) ComputeHash() (*types.Hash, error) {
	lAb, err := block.RpcToLedgerBlock()
	if err != nil {
		return nil, err
	}
	hash := lAb.ComputeHash()
	return &hash, nil
}

func (block *AccountBlock) addExtraInfo(chain chain.Chain) error {

	// TokenInfo
	if block.TokenId != types.ZERO_TOKENID {
		token, _ := chain.GetTokenInfoById(block.TokenId)
		block.TokenInfo = RawTokenInfoToRpc(token, block.TokenId)
	}

	// ReceiveBlockHeight & ReceiveBlockHash
	if block.IsSendBlock() {
		receiveBlock, err := chain.GetReceiveAbBySendAb(block.Hash)
		if err != nil {
			return err
		}
		if receiveBlock != nil {
			heightStr := strconv.FormatUint(receiveBlock.Height, 10)
			block.ReceiveBlockHeight = &heightStr
			block.ReceiveBlockHash = &receiveBlock.Hash
		}
	}

	// ConfirmedTimes & ConfirmedHash
	latestSb := chain.GetLatestSnapshotBlock()
	confirmedBlock, err := chain.GetConfirmSnapshotHeaderByAbHash(block.Hash)
	if err != nil {
		return err
	}
	if confirmedBlock != nil && latestSb != nil && confirmedBlock.Height <= latestSb.Height {
		confirmedTimeStr := strconv.FormatUint(latestSb.Height-confirmedBlock.Height+1, 10)
		block.ConfirmedTimes = &confirmedTimeStr
		block.ConfirmedHash = &confirmedBlock.Hash
		block.Timestamp = confirmedBlock.Timestamp.Unix()
	}
	return nil
}

func ledgerSnapshotBlockToRpcBlock(sb *ledger.SnapshotBlock) (*SnapshotBlock, error) {
	if sb == nil {
		return nil, nil
	}
	rpcBlock := &SnapshotBlock{
		SnapshotBlock: sb,
	}

	rpcBlock.Producer = sb.Producer()

	rpcBlock.Timestamp = sb.Timestamp.Unix()
	return rpcBlock, nil
}

func ledgerToRpcBlock(chain chain.Chain, lAb *ledger.AccountBlock) (*AccountBlock, error) {
	rpcBlock := &AccountBlock{
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

		Producer: lAb.Producer(),
	}
	//height
	rpcBlock.Height = strconv.FormatUint(lAb.Height, 10)

	// Quota & QuotaUsed
	quota := strconv.FormatUint(lAb.Quota, 10)
	rpcBlock.Quota = &quota
	quotaUsed := strconv.FormatUint(lAb.QuotaUsed, 10)
	rpcBlock.QuotaUsed = &quotaUsed

	// FromAddress & ToAddress
	var amount, fee string
	if lAb.IsSendBlock() {
		rpcBlock.FromAddress = lAb.AccountAddress
		rpcBlock.ToAddress = lAb.ToAddress
		if lAb.Amount != nil {
			amount = lAb.Amount.String()
			rpcBlock.Amount = &amount
		}
		if lAb.Fee != nil {
			fee = lAb.Fee.String()
			rpcBlock.Fee = &fee
		}
	} else {
		sendBlock, err := chain.GetAccountBlockByHash(lAb.FromBlockHash)
		if err != nil {
			return nil, err
		}
		if sendBlock != nil {
			rpcBlock.FromAddress = sendBlock.AccountAddress
			rpcBlock.ToAddress = sendBlock.ToAddress

			rpcBlock.FromBlockHash = sendBlock.Hash
			rpcBlock.TokenId = sendBlock.TokenId
			if sendBlock.Amount != nil {
				amount = sendBlock.Amount.String()
				rpcBlock.Amount = &amount
			}
			if sendBlock.Fee != nil {
				fee = sendBlock.Fee.String()
				rpcBlock.Fee = &fee
			}
		}
	}

	// Difficulty & Nonce
	rpcBlock.Nonce = lAb.Nonce
	if lAb.Difficulty != nil {
		difficulty := lAb.Difficulty.String()
		rpcBlock.Difficulty = &difficulty
	}

	if err := rpcBlock.addExtraInfo(chain); err != nil {
		return nil, err
	}

	// SendBlockList
	if len(lAb.SendBlockList) > 0 {
		subBlockList := make([]*AccountBlock, len(lAb.SendBlockList))
		for k, v := range lAb.SendBlockList {
			subRpcTx, err := ledgerToRpcBlock(chain, v)
			if err != nil {
				return nil, err
			}
			subBlockList[k] = subRpcTx
		}
		rpcBlock.SendBlockList = subBlockList
	}

	return rpcBlock, nil
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
	TokenName     string            `json:"tokenName"`
	TokenSymbol   string            `json:"tokenSymbol"`
	TotalSupply   *string           `json:"totalSupply,omitempty"` // *big.Int
	Decimals      uint8             `json:"decimals"`
	Owner         types.Address     `json:"owner"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	MaxSupply     *string           `json:"maxSupply"` // *big.Int
	OwnerBurnOnly bool              `json:"ownerBurnOnly"`
	IsReIssuable  bool              `json:"isReIssuable"`
	Index         uint16            `json:"index"`
}

func RawTokenInfoToRpc(tinfo *types.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:     tinfo.TokenName,
			TokenSymbol:   tinfo.TokenSymbol,
			TotalSupply:   nil,
			Decimals:      tinfo.Decimals,
			Owner:         tinfo.Owner,
			TokenId:       tti,
			OwnerBurnOnly: tinfo.OwnerBurnOnly,
			IsReIssuable:  tinfo.IsReIssuable,
			Index:         tinfo.Index,
		}
		if tinfo.TotalSupply != nil {
			s := tinfo.TotalSupply.String()
			rt.TotalSupply = &s
		}
		if tinfo.MaxSupply != nil {
			s := tinfo.MaxSupply.String()
			rt.MaxSupply = &s
		}
	}
	return rt
}

type TxParam interface {
	LedgerAccountBlock() (*ledger.AccountBlock, error)
}

type NormalRequestRawTxParam struct {
	BlockType byte       `json:"blockType"` // 1
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`
	PublicKey      []byte        `json:"publicKey"`

	ToAddress types.Address     `json:"toAddress"`
	TokenId   types.TokenTypeId `json:"tokenId"`
	Amount    string            `json:"amount"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`
}

func (param NormalRequestRawTxParam) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	if types.IsContractAddr(param.AccountAddress) {
		return nil, errors.New("can't send tx for the contract")
	}

	lAb := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		Hash:           param.Hash,
		PrevHash:       param.PrevHash,
		AccountAddress: param.AccountAddress,
		PublicKey:      param.PublicKey,
		ToAddress:      param.ToAddress,
		TokenId:        param.TokenId,
		Data:           param.Data,

		Signature: param.Signature,
	}

	var err error

	lAb.Height, err = strconv.ParseUint(param.Height, 10, 64)
	if err != nil {
		return nil, err
	}

	lAb.Amount = big.NewInt(0)
	if _, ok := lAb.Amount.SetString(param.Amount, 10); !ok {
		return nil, ErrStrToBigInt
	}

	if param.Nonce != nil {
		if param.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			difficultyStr, ok := new(big.Int).SetString(*param.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = difficultyStr
			lAb.Nonce = param.Nonce
		}
	}
	return lAb, nil
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
